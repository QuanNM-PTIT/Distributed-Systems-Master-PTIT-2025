package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type FriendsHandler struct {
	DB *sql.DB
}

type friendRequestInput struct {
	ToUserID string `json:"toUserId" binding:"required"`
}

type friendAcceptInput struct {
	FromUserID string `json:"fromUserId" binding:"required"`
}

type friendListItem struct {
	UserID string `json:"userId"`
	Username string `json:"username"`
}

type friendRequestItem struct {
	FromUserID string `json:"fromUserId"`
	CreatedAt  int64  `json:"createdAt"`
}

func (h *FriendsHandler) Request(c *gin.Context) {
	var req friendRequestInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	fromUserID := c.GetString("userId")
	if fromUserID == req.ToUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot friend yourself"})
		return
	}
	if !userExists(h.DB, req.ToUserID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if areFriends(h.DB, fromUserID, req.ToUserID) {
		c.JSON(http.StatusConflict, gin.H{"error": "already friends"})
		return
	}
	_, err := h.DB.Exec(`
		INSERT INTO friend_requests (from_user_id, to_user_id, status)
		VALUES (?, ?, 'pending')
		ON DUPLICATE KEY UPDATE status = 'pending'
	`, fromUserID, req.ToUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "request_sent"})
}

func (h *FriendsHandler) Accept(c *gin.Context) {
	var req friendAcceptInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	toUserID := c.GetString("userId")
	if req.FromUserID == toUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE friend_requests
		SET status = 'accepted'
		WHERE from_user_id = ? AND to_user_id = ? AND status = 'pending'
	`, req.FromUserID, toUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}

	_, err = tx.Exec(`
		INSERT IGNORE INTO friends (user_id, friend_user_id) VALUES (?, ?), (?, ?)
	`, req.FromUserID, toUserID, toUserID, req.FromUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

func (h *FriendsHandler) Requests(c *gin.Context) {
	userID := c.GetString("userId")
	rows, err := h.DB.Query(`
		SELECT from_user_id, created_at
		FROM friend_requests
		WHERE to_user_id = ? AND status = 'pending'
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	items := make([]friendRequestItem, 0)
	for rows.Next() {
		var item friendRequestItem
		var createdAt sql.NullTime
		if err := rows.Scan(&item.FromUserID, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if createdAt.Valid {
			item.CreatedAt = createdAt.Time.Unix()
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"requests": items})
}

func (h *FriendsHandler) List(c *gin.Context) {
	userID := c.GetString("userId")
	rows, err := h.DB.Query(`
		SELECT f.friend_user_id, u.username
		FROM friends f
		JOIN users u ON u.user_id = f.friend_user_id
		WHERE f.user_id = ?
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()
	items := make([]friendListItem, 0)
	for rows.Next() {
		var item friendListItem
		if err := rows.Scan(&item.UserID, &item.Username); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"friends": items})
}

func userExists(db *sql.DB, userID string) bool {
	var exists int
	if err := db.QueryRow("SELECT 1 FROM users WHERE user_id = ?", userID).Scan(&exists); err != nil {
		return false
	}
	return true
}

func areFriends(db *sql.DB, userA, userB string) bool {
	var exists int
	err := db.QueryRow(`
		SELECT 1 FROM friends
		WHERE (user_id = ? AND friend_user_id = ?) OR (user_id = ? AND friend_user_id = ?)
		LIMIT 1
	`, userA, userB, userB, userA).Scan(&exists)
	if err == sql.ErrNoRows {
		return false
	}
	return err == nil
}
