package mail

import (
	"context"
	"log"
)

type Message struct {
	To      []string
	Subject string
	HTML    string
	Text    string
}

type Sender interface {
	Send(ctx context.Context, message Message) error
}

type LogSender struct{}

func (LogSender) Send(_ context.Context, message Message) error {
	log.Printf("mail(log): to=%v subject=%q", message.To, message.Subject)
	return nil
}
