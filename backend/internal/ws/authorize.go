package ws

import (
	"database/sql"
)

func (h *Handler) allowedToSignal(from, to, groupID string) (bool, error) {
	if groupID != "" {
		return bothInGroup(h.DB, from, to, groupID)
	}
	return areFriends(h.DB, from, to)
}

func areFriends(db *sql.DB, userA, userB string) (bool, error) {
	var exists int
	err := db.QueryRow(`
		SELECT 1 FROM friends
		WHERE (user_id = ? AND friend_user_id = ?) OR (user_id = ? AND friend_user_id = ?)
		LIMIT 1
	`, userA, userB, userB, userA).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func bothInGroup(db *sql.DB, userA, userB, groupID string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(1) FROM group_members
		WHERE group_id = ? AND user_id IN (?, ?)
	`, groupID, userA, userB).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 2, nil
}
