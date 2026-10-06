package transport

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
	"undermine.ru/undermine-server/internal/protocol"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536
)

type Client struct {
	userID      [16]byte
	conn        *websocket.Conn
	hub         *Hub
	SendChannel chan []byte
}

func NewClient(userID [16]byte, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		userID:      userID,
		conn:        conn,
		hub:         hub,
		SendChannel: make(chan []byte, 256),
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c.userID)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		if messageType == websocket.BinaryMessage {
			if len(data) > 0 && protocol.PacketType(data[0]) == protocol.PacketTypePing {
				pongPacket := []byte{byte(protocol.PacketTypePong)}
				c.SendChannel <- pongPacket
				continue
			}
			c.hub.RouteMessage(c, data)
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.SendChannel:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			n := len(c.SendChannel)
			for i := 0; i < n; i++ {
				_, _ = w.Write(<-c.SendChannel)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
