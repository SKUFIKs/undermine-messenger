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
	old := h.clients[userID]
	h.clients[userID] = client
	h.mu.Unlock()

	// Если для пользователя уже было соединение (например, после повторного
	// входа), закрываем старое, чтобы его ReadPump завершился и не «отписал»
	// новое подключение из хаба.
	if old != nil && old != client && old.conn != nil {
		log.Printf("[Hub] Замена соединения для %x: закрытие старого подключения", userID)
		_ = old.conn.Close()
	}
}

// Unregister удаляет клиента из хаба только если в хабе всё ещё находится
// именно это соединение. Иначе от старого (закрытого) соединения может прийти
// отложенный вызов и удалить уже перерегистрированный новый клиент — тогда
// сообщения этому пользователю будут навсегда уходить в офлайн-очередь.
func (h *Hub) Unregister(userID [16]byte, client *Client) {
	h.mu.Lock()
	if current, ok := h.clients[userID]; ok && current == client {
		delete(h.clients, userID)
	}
	h.mu.Unlock()
}

func (h *Hub) RouteMessage(sender *Client, rawData []byte) {
	packetType := protocol.PacketType(rawData[0])

	if packetType == protocol.PacketTypeGroupMessage {
		h.routeGroupMessage(sender, rawData)
		return
	}

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

	if ok && recipient != nil {
		select {
		case recipient.SendChannel <- rawData:
			ackStatus = 0x02
			log.Printf("[Hub] Сообщение отправлено получателю онлайн")
		default:
			log.Printf("[Hub] Канал получателя полон, сохранение в офлайн-очередь")
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

// routeGroupMessage delivers a group message to all online members (except
// the sender) and queues it for offline members.
func (h *Hub) routeGroupMessage(sender *Client, rawData []byte) {
	packet, err := protocol.DecodeGroupMessagePacket(rawData)
	if err != nil {
		log.Printf("[Hub] Ошибка декодирования группового пакета: %v", err)
		return
	}

	log.Printf("[Hub] Маршрутизация группового сообщения: msgID=%x от %x в группу %x",
		packet.MessageID, sender.userID, packet.GroupID)

	members, err := h.storage.GetGroupMembers(packet.GroupID)
	if err != nil {
		log.Printf("[Hub] Ошибка получения участников группы: %v", err)
		return
	}

	var ackStatus byte

	for _, member := range members {
		if member.UserID == sender.userID {
			continue
		}

		h.mu.RLock()
		recipient, ok := h.clients[member.UserID]
		h.mu.RUnlock()

		if ok && recipient != nil {
			select {
			case recipient.SendChannel <- rawData:
				ackStatus = 0x02
				log.Printf("[Hub] Групповое сообщение отправлено %x онлайн", member.UserID)
			default:
				_ = h.storage.SavePendingMessage(packet.MessageID, member.UserID, rawData)
			}
		} else {
			_ = h.storage.SavePendingMessage(packet.MessageID, member.UserID, rawData)
		}
	}

	if ackStatus == 0 {
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
