package server

type MessageProtocol uint16

type MessagePacket interface {
	SetPayload(payload []byte)
	Deserialize(data []byte) error
	Serialize() []byte
}

type DefaultMessagePacket struct {
	Payload []byte
}

func NewMessagePacket(payload []byte) *DefaultMessagePacket {
	return &DefaultMessagePacket{
		Payload: payload,
	}
}

func (mp *DefaultMessagePacket) SetPayload(payload []byte) {
	mp.Payload = payload
}

func (mp *DefaultMessagePacket) Deserialize(data []byte) error {
	mp.SetPayload(data)

	return nil
}

func (mp *DefaultMessagePacket) Serialize() []byte {
	return mp.Payload
}
