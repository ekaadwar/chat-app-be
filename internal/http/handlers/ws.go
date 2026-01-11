package handlers

import (
	"net/http"
	"strings"

	"chat-be/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"nhooyr.io/websocket"
)

type WSHandler struct {
	Hub                  *ws.Hub
	JWTSecret            string
	WSInsecureSkipVerify bool
}

func (h *WSHandler) Handle(c *gin.Context) {
	// Karena browser native WebSocket sulit set header Authorization,
	// kita pakai query param token=...
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "missing token"})
		return
	}

	userID, err := parseUserIDFromJWT(tokenStr, h.JWTSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}

	opts := &websocket.AcceptOptions{}
	// Default Accept menolak cross-origin. Untuk dev (vite:5173) sering beda origin.
	// InsecureSkipVerify akan mem-bypass verifikasi origin (HANYA untuk dev).
	// Untuk production, sebaiknya pakai OriginPatterns/konfigurasi origin yang benar.
	if h.WSInsecureSkipVerify {
		opts.InsecureSkipVerify = true
	}

	conn, err := websocket.Accept(c.Writer, c.Request, opts)
	if err != nil {
		return // Accept sudah menulis response error
	}

	// Kita tidak butuh menerima data message dari client (push-only),
	// tapi tetap perlu membaca agar control frames (close/ping/pong) diproses.
	conn.CloseRead(c.Request.Context())

	client := h.Hub.AddClient(userID, conn)
	defer h.Hub.RemoveClient(client)

	// block sampai client disconnect
	<-c.Request.Context().Done()
}

func parseUserIDFromJWT(tokenStr, secret string) (uint, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, err
	}

	uidAny := claims["user_id"]
	switch v := uidAny.(type) {
	case float64:
		return uint(v), nil
	case string:
		v = strings.TrimSpace(v)
		// optional parse string → uint, tapi biasanya float64
	}
	return 0, err
}
