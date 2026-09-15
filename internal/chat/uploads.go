package chat

import (
	"database/sql"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Upload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MIME string `json:"mime"`
	Size int64  `json:"size"`
}

func (s *Server) uploadLimits(w http.ResponseWriter, r *http.Request) error {
	p, err := s.policy(r.Context())
	if err != nil {
		return err
	}
	return jsonResponse(w, map[string]int{"avatar_mb": p.AvatarMB, "image_mb": p.ImageMB, "file_mb": p.FileMB})
}
func (s *Server) upload(w http.ResponseWriter, r *http.Request) error {
	user := current(r).ID
	if !s.limits.allow("upload:"+user, 10, 0.2) {
		return fail(429, "上传太频繁，请稍后重试")
	}
	p, err := s.policy(r.Context())
	if err != nil {
		return err
	}
	purpose := r.URL.Query().Get("purpose")
	limit := p.FileMB
	switch purpose {
	case "avatar":
		limit = p.AvatarMB
	case "image":
		limit = p.ImageMB
	case "file":
	default:
		return fail(400, "无效上传类型")
	}
	// Uploads may take longer than ordinary JSON requests on slower connections.
	http.NewResponseController(w).SetReadDeadline(time.Now().Add(3 * time.Minute))
	r.Body = http.MaxBytesReader(w, r.Body, (int64(limit)<<20)+65536)
	reader, err := r.MultipartReader()
	if err != nil {
		return fail(400, "需要上传文件")
	}
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "file" || part.FileName() == "" {
		return fail(400, "需要 file 文件字段")
	}
	defer part.Close()
	name := filepath.Base(strings.ReplaceAll(part.FileName(), "\\", "/"))
	name = strings.Map(func(c rune) rune {
		if c < 32 || c == 127 {
			return -1
		}
		return c
	}, name)
	if len(name) > 240 || name == "" {
		return fail(400, "文件名过长或无效")
	}
	if err = os.MkdirAll(s.store.UploadDir, 0700); err != nil {
		return err
	}
	u := Upload{ID: id(), Name: name}
	path := filepath.Join(s.store.UploadDir, u.ID)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		f.Close()
		if !keep {
			os.Remove(path)
		}
	}()
	u.Size, err = io.Copy(f, io.LimitReader(part, (int64(limit)<<20)+1))
	if err != nil || u.Size > int64(limit)<<20 {
		return fail(413, "文件超过管理员设置的大小上限")
	}
	if u.Size == 0 {
		return fail(400, "不能上传空文件")
	}
	f.Seek(0, io.SeekStart)
	head := make([]byte, 512)
	n, _ := f.Read(head)
	u.MIME = http.DetectContentType(head[:n])
	if purpose == "image" || purpose == "avatar" {
		f.Seek(0, io.SeekStart)
		cfg, kind, err := image.DecodeConfig(f)
		if err != nil || (kind != "png" && kind != "jpeg" && kind != "gif") || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 40000000 {
			return fail(400, "请选择 PNG、JPEG 或 GIF 图片（最多 4000 万像素）")
		}
	} else {
		u.MIME = "application/octet-stream"
	}
	err = s.store.tx(r.Context(), func(tx *sql.Tx) error {
		if _, err := tx.Exec("INSERT INTO uploads VALUES(?,?,?,?,?,?,?)", u.ID, user, u.Name, u.MIME, u.Size, purpose, now()); err != nil {
			return err
		}
		if purpose == "avatar" {
			_, err := tx.Exec("INSERT INTO avatars VALUES(?,?) ON CONFLICT(user_id) DO UPDATE SET upload_id=excluded.upload_id", user, u.ID)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	keep = true
	if purpose == "avatar" {
		s.hub.publish(s.roomPeers(r, user), map[string]string{"type": "avatar", "user_id": user})
	}
	return jsonResponse(w, u)
}
func (s *Server) roomPeers(r *http.Request, user string) []string {
	rows, err := s.store.Read.QueryContext(r.Context(), "SELECT DISTINCT b.user_id FROM members a JOIN members b ON a.room_id=b.room_id WHERE a.user_id=?", user)
	users := []string{user}
	if err != nil {
		return users
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			users = append(users, id)
		}
	}
	return users
}
func (s *Server) avatar(w http.ResponseWriter, r *http.Request) error {
	var id string
	if err := s.store.Read.QueryRowContext(r.Context(), "SELECT upload_id FROM avatars WHERE user_id=?", r.PathValue("id")).Scan(&id); err != nil {
		return fail(404, "暂无头像")
	}
	return s.serveUpload(w, r, id, true)
}
func (s *Server) download(w http.ResponseWriter, r *http.Request) error {
	return s.serveUpload(w, r, r.PathValue("id"), false)
}
func (s *Server) serveUpload(w http.ResponseWriter, r *http.Request, id string, avatar bool) error {
	var u Upload
	var owner string
	err := s.store.Read.QueryRowContext(r.Context(), "SELECT id,name,mime,size,owner FROM uploads WHERE id=?", id).Scan(&u.ID, &u.Name, &u.MIME, &u.Size, &owner)
	if err != nil {
		return fail(404, "文件不存在")
	}
	if !avatar && current(r).Role != "admin" && owner != current(r).ID {
		var n int
		err = s.store.Read.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM message_details d JOIN messages m ON m.id=d.message_id JOIN members b ON b.room_id=m.room_id WHERE d.upload_id=? AND d.recalled_at=0 AND b.user_id=?", id, current(r).ID).Scan(&n)
		if err != nil {
			return err
		}
		if n == 0 {
			return fail(403, "无权访问此文件")
		}
	}
	f, err := os.Open(filepath.Join(s.store.UploadDir, u.ID))
	if err != nil {
		return fail(404, "文件不可用")
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	disposition := "attachment"
	if strings.HasPrefix(u.MIME, "image/") {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", u.MIME)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": u.Name}))
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	http.ServeContent(w, r, u.Name, st.ModTime(), f)
	return nil
}
