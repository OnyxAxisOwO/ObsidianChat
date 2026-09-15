package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Policy struct {
	SiteKey          string `json:"site_key"`
	SecretKey        string `json:"secret_key,omitempty"`
	SecretConfigured bool   `json:"secret_configured"`
	Register         bool   `json:"register"`
	Login            bool   `json:"login"`
	Friend           bool   `json:"friend"`
	Group            bool   `json:"group"`
	AvatarMB         int    `json:"avatar_mb"`
	ImageMB          int    `json:"image_mb"`
	FileMB           int    `json:"file_mb"`
	FriendsMinute    int    `json:"friends_minute"`
	MessagesMinute   int    `json:"messages_minute"`
	ChallengeMinute  int    `json:"challenge_minute"`
	ChallengeHour    int    `json:"challenge_hour"`
}
type challengeActionKey struct{}
type challengeRequired struct{ siteKey, action string }

func (e *challengeRequired) Error() string { return "请完成人机验证" }

func (s *Server) policy(ctx context.Context) (Policy, error) {
	p := Policy{AvatarMB: 2, ImageMB: 10, FileMB: 25, FriendsMinute: 10, MessagesMinute: 120}
	var raw string
	err := s.store.Read.QueryRowContext(ctx, "SELECT value FROM settings WHERE key='policy'").Scan(&raw)
	if err == sql.ErrNoRows {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	err = json.Unmarshal([]byte(raw), &p)
	return p, err
}
func (s *Server) getPolicy(w http.ResponseWriter, r *http.Request) error {
	p, err := s.policy(r.Context())
	if err != nil {
		return err
	}
	p.SecretConfigured = p.SecretKey != ""
	p.SecretKey = ""
	return jsonResponse(w, p)
}
func (s *Server) savePolicy(w http.ResponseWriter, r *http.Request) error {
	var p Policy
	if err := decode(w, r, &p); err != nil {
		return err
	}
	old, err := s.policy(r.Context())
	if err != nil {
		return err
	}
	p.SiteKey = strings.TrimSpace(p.SiteKey)
	p.SecretKey = strings.TrimSpace(p.SecretKey)
	if p.SecretKey == "" {
		p.SecretKey = old.SecretKey
	}
	if len(p.SiteKey) > 256 || len(p.SecretKey) > 256 {
		return fail(400, "密钥过长")
	}
	if p.AvatarMB < 1 || p.AvatarMB > 10 || p.ImageMB < 1 || p.ImageMB > 30 || p.FileMB < 1 || p.FileMB > 100 {
		return fail(400, "头像上限 1–10 MiB，图片 1–30 MiB，文件 1–100 MiB")
	}
	if p.FriendsMinute < 1 || p.FriendsMinute > 200 || p.MessagesMinute < 1 || p.MessagesMinute > 1000 || p.ChallengeMinute < 0 || p.ChallengeMinute > 1000 || p.ChallengeHour < 0 || p.ChallengeHour > 60000 {
		return fail(400, "频率或验证阈值超出范围")
	}
	if (p.Register || p.Login || p.Friend || p.Group || p.ChallengeMinute > 0 || p.ChallengeHour > 0) && (p.SiteKey == "" || p.SecretKey == "") {
		return fail(400, "启用人机验证前请填写站点密钥和服务端密钥")
	}
	p.SecretConfigured = false
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	err = s.store.tx(r.Context(), func(tx *sql.Tx) error {
		if _, err := tx.Exec("INSERT INTO settings(key,value) VALUES('policy',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", string(raw)); err != nil {
			return err
		}
		return audit(tx, current(r).ID, "policy.update", "settings")
	})
	if err != nil {
		return err
	}
	return s.getPolicy(w, r)
}

// A challenge is required again for every message beyond either rolling threshold.
func (s *Server) gate(w http.ResponseWriter, r *http.Request, action string) (bool, error) {
	p, err := s.policy(r.Context())
	if err != nil {
		return false, err
	}
	required := map[string]bool{"register": p.Register, "login": p.Login, "friend": p.Friend, "group": p.Group}[action]
	if action == "message" {
		var minute, hour int
		err = s.store.Read.QueryRowContext(r.Context(), "SELECT COALESCE(SUM(created_at>?),0),COUNT(*) FROM messages WHERE sender=? AND created_at>?", now()-60000, current(r).ID, now()-3600000).Scan(&minute, &hour)
		if err != nil {
			return false, err
		}
		if minute >= p.MessagesMinute {
			return false, fail(429, "已达到每分钟消息上限，请稍后发送")
		}
		required = (p.ChallengeMinute > 0 && minute >= p.ChallengeMinute) || (p.ChallengeHour > 0 && hour >= p.ChallengeHour)
	}
	if required {
		token := r.Header.Get("X-Turnstile-Token")
		if token == "" {
			return false, &challengeRequired{p.SiteKey, action}
		}
		if err = s.verifyTurnstile(r, p, token, action); err != nil {
			return false, err
		}
		*r = *r.WithContext(context.WithValue(r.Context(), challengeActionKey{}, action))
	}
	if action == "friend" || action == "group" {
		limit := p.FriendsMinute
		if action == "group" {
			limit = 10
		}
		err = s.store.tx(r.Context(), func(tx *sql.Tx) error {
			var n int
			if err := tx.QueryRow("SELECT COUNT(*) FROM action_events WHERE user_id=? AND action=? AND created_at>?", current(r).ID, action, now()-60000).Scan(&n); err != nil {
				return err
			}
			if n >= limit {
				return fail(429, "操作次数达到每分钟上限，请稍后再试")
			}
			if _, err := tx.Exec("DELETE FROM action_events WHERE created_at<?", now()-3600000); err != nil {
				return err
			}
			_, err := tx.Exec("INSERT INTO action_events(user_id,action,created_at) VALUES(?,?,?)", current(r).ID, action, now())
			return err
		})
	}
	return err == nil, err
}
func (s *Server) verifyTurnstile(r *http.Request, p Policy, token, action string) error {
	if len(token) > 2048 || p.SecretKey == "" || p.SiteKey == "" {
		return fail(400, "人机验证配置无效")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(url.Values{"secret": {p.SecretKey}, "response": {token}}.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := s.turnstileClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return fail(503, "人机验证暂时不可用，请重试")
	}
	defer res.Body.Close()
	var v struct {
		Success  bool   `json:"success"`
		Action   string `json:"action"`
		Hostname string `json:"hostname"`
	}
	if res.StatusCode != 200 || json.NewDecoder(io.LimitReader(res.Body, 65536)).Decode(&v) != nil {
		return fail(503, "人机验证服务返回异常")
	}
	host := r.Host
	if s.config.Origin != "" {
		u, _ := url.Parse(s.config.Origin)
		if u != nil {
			host = u.Host
		}
	}
	u, _ := url.Parse("https://" + host)
	if !v.Success || v.Action != action || u == nil || !strings.EqualFold(v.Hostname, u.Hostname()) {
		return fail(400, "人机验证未通过或已过期，请重新操作")
	}
	return nil
}
