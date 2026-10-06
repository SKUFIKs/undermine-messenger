package protocol

import (
	"fmt"
)

type PacketType byte

const (
	PacketTypeMessage    PacketType = 0x01
	PacketTypeMessageAck PacketType = 0x02
	PacketTypePing       PacketType = 0x03
	PacketTypePong       PacketType = 0x04
)

const minMessagePacketSize = 45
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
