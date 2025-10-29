package serializepb

import "github.com/suman7383/go-queue/internal/queue"

func FromMessage(msg queue.Message) *Consume {
	return &Consume{
		Id:        msg.ID,
		Payload:   msg.Payload,
		Acked:     msg.Acked,
		Timestamp: msg.Timestamp.String(),
		Retries:   int32(msg.Retries),
	}
}
