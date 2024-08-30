package listener

import (
	"context"
	"errors"
	"log"

	"github.com/segmentio/kafka-go"
)

var (
	ErrFatal = errors.New("fatal error")
)

type (
	ProcessFunc   func(msg []byte) error
	MessageReader interface {
		ReadMessage(ctx context.Context) (kafka.Message, error)
	}
	Listener struct {
		mr MessageReader
	}
)

func New(mr MessageReader) *Listener {
	return &Listener{
		mr: mr,
	}
}

func (l *Listener) Listen(ctx context.Context, pf ProcessFunc) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		message, err := l.mr.ReadMessage(ctx)
		if err != nil {
			return err
		}
		if err := pf(message.Value); err != nil {
			if errors.Is(err, ErrFatal) {
				return err
			}
			log.Printf("got error: %s\n", err.Error())
		}

		return l.Listen(ctx, pf)
	}
}
