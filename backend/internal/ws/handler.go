package ws

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"p2p-chat-app/backend/pkg/utils"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	Hub       *Hub
	DB        *sql.DB
	JWTSecret string
}

type SignalMessage struct {
	Type    string          `json:"type"`
	From    string          `json:"from"`
	To      string          `json:"to"`
	GroupID string          `json:"groupId,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
}

func (h *Handler) ServeWS(c *gin.Context) {
	userID, ok := h.authenticate(c)
	if !ok {
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &Client{UserID: userID, Conn: conn, Send: make(chan []byte, 32)}
	h.Hub.Register(client)
	h.setPresence(userID, "online")
	h.notifyFriendsPresence(userID, "online")

	go client.writeLoop()
	client.readLoop(h)
}

func (h *Handler) authenticate(c *gin.Context) (string, bool) {
	auth := c.GetHeader("Authorization")
	token := ""
	if auth != "" {
		parts := strings.Split(auth, " ")
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token = parts[1]
		}
	}
	if token == "" {
		token = c.Query("token")
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return "", false
	}
	claims, err := utils.ParseToken(token, h.JWTSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return "", false
	}
	return claims.UserID, true
}

func (c *Client) readLoop(h *Handler) {
	defer func() {
		h.Hub.Unregister(c.UserID)
		h.setPresence(c.UserID, "offline")
		h.notifyFriendsPresence(c.UserID, "offline")
		_ = c.Conn.Close()
	}()
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg SignalMessage
		if err := c.Conn.ReadJSON(&msg); err != nil {
			return
		}
		msg.From = c.UserID
		if msg.To == "" || msg.Type == "" {
			c.Send <- []byte(`{"error":"invalid signaling message"}`)
			continue
		}
		if !isAllowedSignalType(msg.Type) {
			c.Send <- []byte(`{"error":"unsupported signaling type"}`)
			continue
		}
		if isGroupSignal(msg.Type) && msg.GroupID == "" {
			c.Send <- []byte(`{"error":"missing groupId for group signaling"}`)
			continue
		}
		allowed, err := h.allowedToSignal(c.UserID, msg.To, msg.GroupID)
		if err != nil {
			c.Send <- []byte(`{"error":"authorization failed"}`)
			continue
		}
		if !allowed {
			c.Send <- []byte(`{"error":"not allowed"}`)
			continue
		}
		if ok := h.Hub.Send(msg.To, msg); !ok {
			c.Send <- []byte(`{"error":"target offline"}`)
		}
	}
}

func (c *Client) writeLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			_ = c.Conn.WriteMessage(websocket.TextMessage, msg)
		case <-ticker.C:
			_ = c.Conn.WriteMessage(websocket.PingMessage, []byte("ping"))
		}
	}
}
