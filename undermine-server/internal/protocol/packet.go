package protocol

import (
	"fmt"
)

type PacketType byte

const (
	PacketTypeMessage      PacketType = 0x01
	PacketTypeMessageAck   PacketType = 0x02
	PacketTypePing         PacketType = 0x03
	PacketTypePong         PacketType = 0x04
	PacketTypeGroupMessage PacketType = 0x05
)

const minMessagePacketSize = 45
const minGroupPacketSize = 61
const ackPacketSize = 18

type MessagePacket struct {
	Type        PacketType
	MessageID   [16]byte
	RecipientID [16]byte
	Nonce       [12]byte
	Payload     []byte
}

type AckPacket struct {
	Type      PacketType
	MessageID [16]byte
	Status    byte
}

// GroupMessagePacket is an encrypted message sent to a group.
// Layout: type(1) + senderID(16) + groupID(16) + messageID(16) + nonce(12) + payload
type GroupMessagePacket struct {
	Type      PacketType
	SenderID  [16]byte
	GroupID   [16]byte
	MessageID [16]byte
	Nonce     [12]byte
	Payload   []byte
}

func DecodeMessagePacket(data []byte) (*MessagePacket, error) {
	if len(data) < minMessagePacketSize {
		return nil, fmt.Errorf("пакет поврежден или слишком короткий")
	}

	var packet MessagePacket
	packet.Type = PacketType(data[0])
	copy(packet.MessageID[:], data[1:17])
	copy(packet.RecipientID[:], data[17:33])
	copy(packet.Nonce[:], data[33:45])
	packet.Payload = data[45:]

	return &packet, nil
}

func EncodeMessagePacket(packet *MessagePacket) []byte {
	buffer := make([]byte, minMessagePacketSize+len(packet.Payload))
	buffer[0] = byte(packet.Type)
	copy(buffer[1:17], packet.MessageID[:])
	copy(buffer[17:33], packet.RecipientID[:])
	copy(buffer[33:45], packet.Nonce[:])
	copy(buffer[45:], packet.Payload)
	return buffer
}

func EncodeAckPacket(packet *AckPacket) []byte {
	buffer := make([]byte, ackPacketSize)
	buffer[0] = byte(packet.Type)
	copy(buffer[1:17], packet.MessageID[:])
	buffer[17] = packet.Status
	return buffer
}

func DecodeAckPacket(data []byte) (*AckPacket, error) {
	if len(data) < ackPacketSize {
		return nil, fmt.Errorf("размер ACK пакета слишком мал: %d (ожидается %d)", len(data), ackPacketSize)
	}

	var packet AckPacket
	packet.Type = PacketType(data[0])
	copy(packet.MessageID[:], data[1:17])
	packet.Status = data[17]

	return &packet, nil
}

func DecodeGroupMessagePacket(data []byte) (*GroupMessagePacket, error) {
	if len(data) < minGroupPacketSize {
		return nil, fmt.Errorf("групповой пакет повреждён или слишком короткий")
	}

	var packet GroupMessagePacket
	packet.Type = PacketType(data[0])
	copy(packet.SenderID[:], data[1:17])
	copy(packet.GroupID[:], data[17:33])
	copy(packet.MessageID[:], data[33:49])
	copy(packet.Nonce[:], data[49:61])
	packet.Payload = data[61:]

	return &packet, nil
}

func EncodeGroupMessagePacket(packet *GroupMessagePacket) []byte {
	buffer := make([]byte, minGroupPacketSize+len(packet.Payload))
	buffer[0] = byte(packet.Type)
	copy(buffer[1:17], packet.SenderID[:])
	copy(buffer[17:33], packet.GroupID[:])
	copy(buffer[33:49], packet.MessageID[:])
	copy(buffer[49:61], packet.Nonce[:])
	copy(buffer[61:], packet.Payload)
	return buffer
}
