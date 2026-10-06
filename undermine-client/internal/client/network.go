package client

import (
	"crypto/rand"
	"fmt"

	"github.com/gorilla/websocket"
	"undermine.ru/undermine-messenger/internal/protocol"
)

// NetworkClient представляет WebSocket клиент для обмена сообщениями
type NetworkClient struct {
	conn       *websocket.Conn
	userID     [16]byte
	sharedKeys map[[16]byte][32]byte
}

// NewNetworkClient создает новый сетевой клиент
func NewNetworkClient(userID [16]byte) *NetworkClient {
	return &NetworkClient{
		conn:       nil,
		userID:     userID,
		sharedKeys: make(map[[16]byte][32]byte),
	}
}

// SetContactSharedKey устанавливает общий ключ для контакта
func (c *NetworkClient) SetContactSharedKey(contactID [16]byte, sharedKey [32]byte) {
	c.sharedKeys[contactID] = sharedKey
}

// Connect подключается к серверу через WebSocket
func (c *NetworkClient) Connect(serverURL string) error {
	// Преобразуем HTTP URL в WebSocket URL
	wsURL := serverURL
	if len(wsURL) > 5 && wsURL[:5] == "https" {
		wsURL = "wss" + wsURL[5:]
	} else if len(wsURL) > 4 && wsURL[:4] == "http" {
		wsURL = "ws" + wsURL[4:]
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// SendTextMessage отправляет текстовое сообщение
func (c *NetworkClient) SendTextMessage(recipientID [16]byte, text string) ([16]byte, error) {
	sharedKey, ok := c.sharedKeys[recipientID]
	if !ok {
		return [16]byte{}, fmt.Errorf("нет ключа шифрования для контакта")
	}

	var msgID [16]byte
	if _, err := rand.Read(msgID[:]); err != nil {
		return msgID, err
	}

	nonce, ciphertext, err := Encrypt(text, sharedKey)
	if err != nil {
		return msgID, err
	}

	packet := protocol.MessagePacket{
		Type:        protocol.PacketTypeMessage,
		MessageID:   msgID,
		RecipientID: recipientID,
		Nonce:       nonce,
		Payload:     ciphertext,
	}

	rawBytes := protocol.EncodeMessagePacket(&packet)

	err = c.conn.WriteMessage(websocket.BinaryMessage, rawBytes)
	return msgID, err
}

// Listen начинает прослушивание входящих сообщений
func (c *NetworkClient) Listen(onMessage func(msgID [16]byte, senderID [16]byte, text string), onAck func(msgID [16]byte, status byte)) {
	for {
		_, rawBytes, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		if len(rawBytes) == 0 {
			continue
		}

		packetType := protocol.PacketType(rawBytes[0])

		switch packetType {
		case protocol.PacketTypeMessage:
			packet, err := protocol.DecodeMessagePacket(rawBytes)
			if err == nil {
				for contactID, key := range c.sharedKeys {
					text, decErr := Decrypt(packet.Nonce, packet.Payload, key)
					if decErr == nil {
						onMessage(packet.MessageID, contactID, text)

						ack := protocol.AckPacket{
							Type:      protocol.PacketTypeMessageAck,
							MessageID: packet.MessageID,
							Status:    0x02,
						}
						_ = c.conn.WriteMessage(websocket.BinaryMessage, protocol.EncodeAckPacket(&ack))
						break
					}
				}
			}

		case protocol.PacketTypeMessageAck:
			ack, err := protocol.DecodeAckPacket(rawBytes)
			if err == nil {
				onAck(ack.MessageID, ack.Status)
			}
		}
	}
}

// Close закрывает соединение
func (c *NetworkClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
