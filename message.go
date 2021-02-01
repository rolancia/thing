package server

import (
	"encoding/binary"
)

type MessageProtocol uint16

type messagePacket interface {
	SetProtocol(proto MessageProtocol)
	SetPayload(payload []byte)
	Deserialize(data []byte) error
	Serialize() []byte
}

type DefaultMessagePacket struct {
	Protocol MessageProtocol
	Payload  []byte
}

func NewMessagePacket(proto MessageProtocol, payload []byte) *DefaultMessagePacket {
	return &DefaultMessagePacket{
		Protocol: proto,
		Payload:  payload,
	}
}

func (mp *DefaultMessagePacket) SetProtocol(proto MessageProtocol) {
	mp.Protocol = proto
}

func (mp *DefaultMessagePacket) SetPayload(payload []byte) {
	mp.Payload = payload
}

func (mp *DefaultMessagePacket) Deserialize(data []byte) error {
	if len(data) < 2 {
		return ErrNoProtocol
	}

	mp.SetProtocol(MessageProtocol(binary.BigEndian.Uint16(data[:2])))
	mp.SetPayload(data[2:])

	return nil
}

func (mp *DefaultMessagePacket) Serialize() []byte {
	proto := make([]byte, 2, 2)
	binary.BigEndian.PutUint16(proto, uint16(mp.Protocol))

	buffer := make([]byte, 0, len(proto)+len(mp.Payload))
	buffer = append(buffer, proto...)
	buffer = append(buffer, mp.Payload...)

	return buffer
}
