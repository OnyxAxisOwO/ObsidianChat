package chat

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func sendTest(t *testing.T, f *fixture, c *http.Cookie, room string, body any) Message {
	t.Helper()
	var m Message
	data := f.request(t, c, "POST", "/api/rooms/"+room+"/messages", body, 200)
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}
func TestReplyForwardRecallAndAdminRecords(t *testing.T) {
	f := newFixture(t)
	a, ca := f.account(t, "admin", true)
	b, cb := f.account(t, "alice", false)
	c, cc := f.account(t, "bravo", false)
	ab := f.befriend(t, a, b, ca, cb)
	ac := f.befriend(t, a, c, ca, cc)
	m := sendTest(t, f, ca, ab, map[string]any{"body": "original", "client_id": "original-1"})
	reply := sendTest(t, f, cb, ab, map[string]any{"body": "reply", "client_id": "reply-0001", "reply_to": m.ID})
	if reply.Reply == nil || reply.Reply.Body != "original" {
		t.Fatal("missing reply")
	}
	f.request(t, cc, "POST", "/api/rooms/"+ac+"/messages", map[string]any{"body": "", "client_id": "private-fw", "forward_from": m.ID}, 403)
	forwarded := sendTest(t, f, ca, ac, map[string]any{"body": "", "client_id": "forward-01", "forward_from": m.ID})
	if forwarded.Body != "original" || forwarded.ForwardFrom != m.ID {
		t.Fatal("forward lost content")
	}
	path := "/api/rooms/" + ab + "/messages/" + jsonNumber(m.ID)
	f.request(t, cb, "DELETE", path, nil, 403)
	f.request(t, ca, "DELETE", path, nil, 200)
	var history []Message
	json.Unmarshal(f.request(t, cb, "GET", "/api/rooms/"+ab+"/messages", nil, 200), &history)
	if history[0].RecalledAt == 0 || history[0].Body != "[消息已撤回]" || history[1].Reply.Body != "[消息已撤回]" {
		t.Fatal("recall not propagated to history/reply")
	}
	f.request(t, ca, "POST", "/api/rooms/"+ac+"/messages", map[string]any{"body": "", "client_id": "recalled-fw", "forward_from": m.ID}, 400)
	f.request(t, cb, "GET", "/api/admin/messages", nil, 403)
	var records []Message
	json.Unmarshal(f.request(t, ca, "GET", "/api/admin/messages?user="+b.ID, nil, 200), &records)
	if len(records) != 2 {
		t.Fatalf("wrong user records %d", len(records))
	}
	var count int
	f.store.Read.QueryRow("SELECT COUNT(*) FROM audit WHERE action='messages.view'").Scan(&count)
	if count != 1 {
		t.Fatal("missing access audit")
	}
}
func jsonNumber(n int64) string { b, _ := json.Marshal(n); return string(b) }
func uploadTest(t *testing.T, f *fixture, c *http.Cookie, purpose, name string, body []byte, status int) Upload {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", name)
	part.Write(body)
	mw.Close()
	req, _ := http.NewRequest("POST", f.http.URL+"/api/uploads?purpose="+purpose, &buf)
	req.AddCookie(c)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != status {
		t.Fatalf("upload %d want %d: %s", res.StatusCode, status, raw)
	}
	var u Upload
	json.Unmarshal(raw, &u)
	return u
}
func TestUploadsLimitsAndAccess(t *testing.T) {
	f := newFixture(t)
	a, ca := f.account(t, "admin", true)
	b, cb := f.account(t, "alice", false)
	_, cc := f.account(t, "bravo", false)
	room := f.befriend(t, a, b, ca, cb)
	p, _ := f.app.policy(context.Background())
	p.FileMB = 1
	p.AvatarMB = 1
	p.ImageMB = 1
	f.request(t, ca, "PATCH", "/api/admin/policy", p, 200)
	uploadTest(t, f, cb, "avatar", "fake.png", []byte("not an image"), 400)
	uploadTest(t, f, cb, "file", "large.bin", make([]byte, 1048577), 413)
	var pngData bytes.Buffer
	im := image.NewRGBA(image.Rect(0, 0, 2, 2))
	im.Set(0, 0, color.White)
	png.Encode(&pngData, im)
	avatar := uploadTest(t, f, cb, "avatar", "avatar.png", pngData.Bytes(), 200)
	if avatar.MIME != "image/png" {
		t.Fatal("avatar mime")
	}
	f.request(t, ca, "GET", "/api/avatars/"+b.ID, nil, 200)
	file := uploadTest(t, f, cb, "file", "notes.txt", []byte("private attachment"), 200)
	f.request(t, cc, "GET", "/api/uploads/"+file.ID, nil, 403)
	m := sendTest(t, f, cb, room, map[string]any{"body": "", "client_id": "attach-001", "upload_id": file.ID})
	if m.Attachment == nil || m.Attachment.ID != file.ID {
		t.Fatal("missing attachment")
	}
	f.request(t, ca, "GET", "/api/uploads/"+file.ID, nil, 200)
	f.request(t, cc, "GET", "/api/uploads/"+file.ID, nil, 403)
	f.request(t, ca, "POST", "/api/rooms/"+room+"/messages", map[string]any{"body": "stolen", "client_id": "attach-own", "upload_id": file.ID}, 400)
	f.request(t, cb, "DELETE", "/api/rooms/"+room+"/messages/"+jsonNumber(m.ID), nil, 200)
}

type transportFunc func(*http.Request) (*http.Response, error)

func (fn transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }
func TestPolicyTurnstileAndMessageThresholds(t *testing.T) {
	f := newFixture(t)
	a, ca := f.account(t, "admin", true)
	b, cb := f.account(t, "alice", false)
	room := f.befriend(t, a, b, ca, cb)
	p, _ := f.app.policy(context.Background())
	p.SiteKey = "public-key"
	p.SecretKey = "private-test-key"
	p.Register = true
	p.Login = true
	p.Friend = true
	p.Group = true
	p.ChallengeMinute = 1
	p.MessagesMinute = 2
	raw := f.request(t, ca, "PATCH", "/api/admin/policy", p, 200)
	if bytes.Contains(raw, []byte("private-test-key")) {
		t.Fatal("secret leaked")
	}
	for _, path := range []string{"/api/register", "/api/login", "/api/friends/requests", "/api/rooms"} {
		f.request(t, ca, "POST", path, map[string]any{}, 428)
	}
	sendTest(t, f, cb, room, map[string]any{"body": "first", "client_id": "rate-first"})
	payload := map[string]any{"body": "challenge", "client_id": "rate-second"}
	f.request(t, cb, "POST", "/api/rooms/"+room+"/messages", payload, 428)
	var count int
	f.store.Read.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
	if count != 1 {
		t.Fatal("challenge caused a write")
	}
	f.app.turnstileClient = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		r.ParseForm()
		if r.Form.Get("secret") != "private-test-key" {
			t.Fatal("wrong verification secret")
		}
		data := `{"success":true,"action":"message","hostname":"127.0.0.1"}`
		if r.Form.Get("response") == "bad" {
			data = `{"success":true,"action":"login","hostname":"127.0.0.1"}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(data)), Header: make(http.Header)}, nil
	})}
	for _, token := range []string{"bad", "valid"} {
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", f.http.URL+"/api/rooms/"+room+"/messages", bytes.NewReader(body))
		req.AddCookie(cb)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Turnstile-Token", token)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		want := 200
		if token == "bad" {
			want = 400
		}
		if res.StatusCode != want {
			t.Fatalf("verification %s = %d", token, res.StatusCode)
		}
	}
	f.request(t, cb, "POST", "/api/rooms/"+room+"/messages", map[string]any{"body": "too many", "client_id": "rate-third"}, 429)
	f.request(t, cb, "POST", "/api/rooms/"+room+"/messages", payload, 200)
	// Hour threshold remains effective after minute counts expire.
	f.store.Write.Exec("UPDATE messages SET created_at=?", now()-120000)
	p.ChallengeMinute = 0
	p.ChallengeHour = 1
	f.request(t, ca, "PATCH", "/api/admin/policy", p, 200)
	f.request(t, cb, "POST", "/api/rooms/"+room+"/messages", map[string]any{"body": "hour check", "client_id": "rate-hour"}, 428)
}
func TestFriendRequestRatePersists(t *testing.T) {
	f := newFixture(t)
	_, ca := f.account(t, "admin", true)
	_, cb := f.account(t, "alice", false)
	c, _ := f.account(t, "bravo", false)
	d, _ := f.account(t, "charlie", false)
	p, _ := f.app.policy(context.Background())
	p.FriendsMinute = 1
	f.request(t, ca, "PATCH", "/api/admin/policy", p, 200)
	f.request(t, cb, "POST", "/api/friends/requests", map[string]string{"user_id": c.ID}, 200)
	f.request(t, cb, "POST", "/api/friends/requests", map[string]string{"user_id": d.ID}, 429)
}
func TestUpgradeFromTextOnlySchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	legacy := strings.Split(schema, "CREATE TABLE IF NOT EXISTS uploads")[0]
	if _, err = db.Exec(legacy); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO users(id,username,name,password,created_at) VALUES('u','user','User','hash',1);
 INSERT INTO rooms(id,kind,name,owner,created_at) VALUES('r','group','Original','u',1);
 INSERT INTO members(room_id,user_id) VALUES('r','u');
 INSERT INTO messages(room_id,sender,body,client_id,created_at) VALUES('r','u','historical message','legacy-01',1);`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	m, err := scanMessage(store.Read.QueryRow(messageSelect + " WHERE m.id=1"))
	if err != nil || m.Body != "historical message" || m.RecalledAt != 0 || m.Attachment != nil {
		t.Fatalf("legacy history changed: %+v %v", m, err)
	}
}
func TestAttachmentAccessAfterRecall(t *testing.T) {
	f := newFixture(t)
	_, _ = f.account(t, "admin", true)
	a, ca := f.account(t, "alice", false)
	b, cb := f.account(t, "bravo", false)
	room := f.befriend(t, a, b, ca, cb)
	file := uploadTest(t, f, ca, "file", "private.txt", []byte("test"), 200)
	m := sendTest(t, f, ca, room, map[string]any{"body": "", "upload_id": file.ID, "client_id": "access-01"})
	f.request(t, cb, "GET", "/api/uploads/"+file.ID, nil, 200)
	f.request(t, ca, "DELETE", "/api/rooms/"+room+"/messages/"+jsonNumber(m.ID), nil, 200)
	f.request(t, cb, "GET", "/api/uploads/"+file.ID, nil, 403)
}
