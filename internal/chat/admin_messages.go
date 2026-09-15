package chat

import (
	"database/sql"
	"net/http"
	"strings"
)

func (s *Server) adminMessages(w http.ResponseWriter, r *http.Request) error {
	user, room, q := r.URL.Query().Get("user"), r.URL.Query().Get("room"), strings.TrimSpace(r.URL.Query().Get("q"))
	if len(user) > 64 || len(room) > 64 || len(q) > 200 {
		return fail(400, "搜索条件过长")
	}
	// Record access, including searches with no matches; never log message content.
	if err := s.store.tx(r.Context(), func(tx *sql.Tx) error { return audit(tx, current(r).ID, "messages.view", "user="+user+" room="+room) }); err != nil {
		return err
	}
	rows, err := s.store.Read.QueryContext(r.Context(), messageSelect+` WHERE (?='' OR m.sender=? OR EXISTS(SELECT 1 FROM members b WHERE b.room_id=m.room_id AND b.user_id=?)) AND (?='' OR m.room_id=?) AND (?='' OR m.body LIKE ?) ORDER BY m.id DESC LIMIT 50 OFFSET ?`, user, user, user, room, room, q, "%"+q+"%", pageOffset(r))
	if err != nil {
		return err
	}
	defer rows.Close()
	messages := []Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return err
		}
		messages = append(messages, m)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return jsonResponse(w, messages)
}
