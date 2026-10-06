package transport

import (
	"log"
	"sync"

	"undermine.ru/undermine-server/internal/protocol"
	"undermine.ru/undermine-server/internal/storage"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[[16]byte]*Client

	storage *storage.SQLiteStorage
}

func NewHub(storage *storage.SQLiteStorage) *Hub {
	return &Hub{
		clients: make(map[[16]byte]*Client),
		storage: storage,
	}
}

func (h *Hub) Register(userID [16]byte, client *Client) {
	h.mu.Lock()
	h.clients[userID] = client
	h.mu.Unlock()
}

func (h *Hub) Unregister(userID [16]byte) {
	h.mu.Lock()
	delete(h.clients, userID)
	h.mu.Unlock()
}

func (h *Hub) RouteMessage(sender *Client, rawData []byte) {

	packet, err := protocol.DecodeMessagePacket(rawData)

	if err != nil {
		log.Printf("[Hub] Ошибка декодирования пакета: %v", err)
		return
	}

	log.Printf("[Hub] Маршрутизация сообщения: msgID=%x от %x к %x",
		packet.MessageID, sender.userID, packet.RecipientID)

	h.mu.RLock()

	recipient, ok := h.clients[packet.RecipientID]

	h.mu.RUnlock()

	var ackStatus byte

	if ok == true && recipient != nil {
		log.Printf("[Hub] Получатель онлайн, отправка в канал")
		select {
		case recipient.SendChannel <- rawData:
			ackStatus = 0x02
			log.Printf("[Hub] Сообщение отправлено получателю")

		default:
			log.Printf("[Hub] Канал получателя заполнен, сохранение в офлайн-очередь")
			_ = h.storage.SavePendingMessage(packet.MessageID, packet.RecipientID, rawData)
			ackStatus = 0x01
		}

	} else {
		log.Printf("[Hub] Получатель офлайн, сохранение в офлайн-очередь")
		_ = h.storage.SavePendingMessage(packet.MessageID, packet.RecipientID, rawData)
		ackStatus = 0x01
	}

	ackPacket := protocol.AckPacket{
		Type:      protocol.PacketTypeMessageAck,
		MessageID: packet.MessageID,
		Status:    ackStatus,
	}

	ackBytes := protocol.EncodeAckPacket(&ackPacket)

	sender.SendChannel <- ackBytes

}
