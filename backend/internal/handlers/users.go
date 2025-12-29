package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type UsersHandler struct {
	DB *sql.DB
}

type userSearchItem struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

type userMeResponse struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

func (h *UsersHandler) Me(c *gin.Context) {
	userID := c.GetString("userId")
	var username string
	err := h.DB.QueryRow(`SELECT username FROM users WHERE user_id = ?`, userID).Scan(&username)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, userMeResponse{UserID: userID, Username: username})
}

func (h *UsersHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("query"))
	if len(query) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query too short"})
		return
	}
	userID := c.GetString("userId")
	like := "%%" + query + "%%"
	rows, err := h.DB.Query(`
		SELECT user_id, username
		FROM users
		WHERE username LIKE ? AND user_id <> ?
		ORDER BY username ASC
		LIMIT 10
	`, like, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	items := make([]userSearchItem, 0)
	for rows.Next() {
		var item userSearchItem
		if err := rows.Scan(&item.UserID, &item.Username); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"users": items})
}
