package client

import (
	"time"

	"undermine.ru/undermine-messenger/internal/protocol"
)

// StartPing запускает периодическую отправку ping-пакетов
func (c *NetworkClient) StartPing(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if c.conn == nil {
				return
			}

			pingPacket := []byte{byte(protocol.PacketTypePing)}
			if err := c.conn.WriteMessage(1, pingPacket); err != nil {
				return
			}
		}
	}()
}
