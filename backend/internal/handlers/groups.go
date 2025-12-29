package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GroupsHandler struct {
	DB *sql.DB
}

type createGroupInput struct {
	Name string `json:"name" binding:"required,min=2,max=64"`
}

type inviteGroupInput struct {
	GroupID string `json:"groupId" binding:"required"`
	UserID  string `json:"userId" binding:"required"`
}

type leaveGroupInput struct {
	GroupID string `json:"groupId" binding:"required"`
}

type groupMemberItem struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
	Username string `json:"username"`
}

type groupListItem struct {
	GroupID     string `json:"groupId"`
	Name        string `json:"name"`
	OwnerUserID string `json:"ownerUserId"`
}

func (h *GroupsHandler) Create(c *gin.Context) {
	var req createGroupInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	ownerID := c.GetString("userId")
	groupID := uuid.NewString()

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO `+"`groups`"+` (group_id, name, owner_user_id) VALUES (?, ?, ?)`, groupID, req.Name, ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	_, err = tx.Exec(`INSERT INTO `+"`group_members`"+` (group_id, user_id, role) VALUES (?, ?, 'owner')`, groupID, ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"groupId": groupID})
}

func (h *GroupsHandler) Invite(c *gin.Context) {
	var req inviteGroupInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	inviterID := c.GetString("userId")
	if !userExists(h.DB, req.UserID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if !isGroupMember(h.DB, req.GroupID, inviterID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a group member"})
		return
	}
	if isGroupMember(h.DB, req.GroupID, req.UserID) {
		c.JSON(http.StatusConflict, gin.H{"error": "already in group"})
		return
	}
	_, err := h.DB.Exec(`INSERT INTO `+"`group_members`"+` (group_id, user_id, role) VALUES (?, ?, 'member')`, req.GroupID, req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "invited"})
}

func (h *GroupsHandler) Leave(c *gin.Context) {
	var req leaveGroupInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	userID := c.GetString("userId")
	if !isGroupMember(h.DB, req.GroupID, userID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not a member"})
		return
	}

	ownerID, err := groupOwner(h.DB, req.GroupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if ownerID == userID {
		memberCount, err := groupMemberCount(h.DB, req.GroupID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if memberCount > 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "owner must transfer before leaving"})
			return
		}
	}

	_, err = h.DB.Exec(`DELETE FROM `+"`group_members`"+` WHERE group_id = ? AND user_id = ?`, req.GroupID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "left"})
}

func (h *GroupsHandler) Members(c *gin.Context) {
	groupID := c.Param("id")
	userID := c.GetString("userId")
	if !isGroupMember(h.DB, groupID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a group member"})
		return
	}

	rows, err := h.DB.Query(`
		SELECT gm.user_id, gm.role, u.username
		FROM `+"`group_members`"+` gm
		JOIN users u ON u.user_id = gm.user_id
		WHERE gm.group_id = ?
		ORDER BY gm.created_at ASC
	`, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()
	members := make([]groupMemberItem, 0)
	for rows.Next() {
		var item groupMemberItem
		if err := rows.Scan(&item.UserID, &item.Role, &item.Username); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		members = append(members, item)
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

func (h *GroupsHandler) List(c *gin.Context) {
	userID := c.GetString("userId")
	rows, err := h.DB.Query(`
		SELECT g.group_id, g.name, g.owner_user_id
		FROM `+"`groups`"+` g
		JOIN `+"`group_members`"+` gm ON gm.group_id = g.group_id
		WHERE gm.user_id = ?
		ORDER BY g.created_at DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	items := make([]groupListItem, 0)
	for rows.Next() {
		var item groupListItem
		if err := rows.Scan(&item.GroupID, &item.Name, &item.OwnerUserID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"groups": items})
}

func isGroupMember(db *sql.DB, groupID, userID string) bool {
	var exists int
	if err := db.QueryRow(`SELECT 1 FROM `+"`group_members`"+` WHERE group_id = ? AND user_id = ?`, groupID, userID).Scan(&exists); err != nil {
		return false
	}
	return true
}

func groupOwner(db *sql.DB, groupID string) (string, error) {
	var ownerID string
	err := db.QueryRow(`SELECT owner_user_id FROM `+"`groups`"+` WHERE group_id = ?`, groupID).Scan(&ownerID)
	return ownerID, err
}

func groupMemberCount(db *sql.DB, groupID string) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(1) FROM `+"`group_members`"+` WHERE group_id = ?`, groupID).Scan(&count)
	return count, err
}
