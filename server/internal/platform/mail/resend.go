package mail

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v3"
)

type ResendSender struct {
	client *resend.Client
	from   string
}

func NewResendSender(apiKey, from string) ResendSender {
	return ResendSender{client: resend.NewClient(apiKey), from: from}
}

func (s ResendSender) Send(ctx context.Context, message Message) error {
	_ = ctx // Resend SDK Send currently has no context parameter.
	_, err := s.client.Emails.Send(&resend.SendEmailRequest{
		From:    s.from,
		To:      message.To,
		Subject: message.Subject,
		Html:    message.HTML,
		Text:    message.Text,
	})
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}
	return nil
}
