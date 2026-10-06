package transport

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
	"undermine.ru/undermine-server/internal/protocol"
)

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	maxMessageSize = 1024 * 1024

	sendChannelCapacity = 256
)

type Client struct {
	userID      [16]byte
	conn        *websocket.Conn
	SendChannel chan []byte
	hub         *Hub
}

func NewClient(userID [16]byte, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		userID:      userID,
		conn:        conn,
		SendChannel: make(chan []byte, sendChannelCapacity),
		hub:         hub,
	}
}

func (c *Client) ReadPump() {

	defer func() {
		log.Printf("[Client] Отключение клиента: userID=%x", c.userID)
		c.hub.Unregister(c.userID)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("[Client] Ошибка чтения от %x: %v", c.userID, err)
			break
		}

		c.conn.SetReadDeadline(time.Now().Add(pongWait))

		if len(message) == 0 {
			continue
		}
		packetType := protocol.PacketType(message[0])

		log.Printf("[Client] Получен пакет от %x: type=%d size=%d", c.userID, packetType, len(message))

		switch packetType {
		case protocol.PacketTypeMessage:
			c.hub.RouteMessage(c, message)

		case protocol.PacketTypeMessageAck:
			if len(message) >= 17 {
				var msgID [16]byte
				copy(msgID[:], message[1:17])
				log.Printf("[Client] ACK от %x: msgID=%x", c.userID, msgID)
				_ = c.hub.storage.DeletePendingMessage(msgID)
			}
		case protocol.PacketTypePing:
			log.Printf("[Client] Ping от %x", c.userID)
			c.SendChannel <- []byte{byte(protocol.PacketTypePong)}
		}

	}
}

func (c *Client) WritePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		message, ok := <-c.SendChannel

		if !ok {

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))

		err := c.conn.WriteMessage(websocket.BinaryMessage, message)

		if err != nil {
			return
		}
	}
}
