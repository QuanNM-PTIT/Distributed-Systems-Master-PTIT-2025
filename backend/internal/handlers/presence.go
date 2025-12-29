package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PresenceHandler struct {
	DB *sql.DB
}

type presenceItem struct {
	UserID string `json:"userId"`
	Status string `json:"status"`
	LastSeen *int64 `json:"lastSeen"`
}

func (h *PresenceHandler) List(c *gin.Context) {
	userID := c.GetString("userId")
	rows, err := h.DB.Query(`
		SELECT f.friend_user_id, COALESCE(p.status, 'offline'), p.last_seen
		FROM friends f
		LEFT JOIN presence p ON p.user_id = f.friend_user_id
		WHERE f.user_id = ?
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	items := make([]presenceItem, 0)
	for rows.Next() {
		var item presenceItem
		var lastSeen sql.NullTime
		if err := rows.Scan(&item.UserID, &item.Status, &lastSeen); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if lastSeen.Valid {
			ts := lastSeen.Time.Unix()
			item.LastSeen = &ts
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"presence": items})
}
