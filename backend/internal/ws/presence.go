package ws

import (
	"database/sql"
	"encoding/json"
	"log"
)

type presencePayload struct {
	UserID string `json:"userId"`
	Status string `json:"status"`
}

func (h *Handler) setPresence(userID, status string) {
	_, err := h.DB.Exec(`
		INSERT INTO presence (user_id, status, last_seen)
		VALUES (?, ?, NOW())
		ON DUPLICATE KEY UPDATE status = VALUES(status), last_seen = NOW()
	`, userID, status)
	if err != nil {
		log.Printf("presence update error: %v", err)
	}
}

func (h *Handler) notifyFriendsPresence(userID, status string) {
	friends, err := friendsOf(h.DB, userID)
	if err != nil {
		log.Printf("presence notify error: %v", err)
		return
	}
	payload, _ := json.Marshal(presencePayload{UserID: userID, Status: status})
	msg := SignalMessage{
		Type:    "presence.update",
		From:    userID,
		Payload: payload,
	}
	for _, friendID := range friends {
		msg.To = friendID
		h.Hub.Send(friendID, msg)
	}
}

func friendsOf(db *sql.DB, userID string) ([]string, error) {
	rows, err := db.Query(`SELECT friend_user_id FROM friends WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
