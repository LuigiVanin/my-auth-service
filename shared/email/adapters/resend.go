package adapters

import (
	"auth_service/shared/email"
	"errors"

	"github.com/resend/resend-go/v3"
)

type ResendAdapter struct {
	email.BaseAdapter
}

var _ email.IEmailAdapter = &ResendAdapter{}

type ResendOptions struct {
	ApiKey string
}

func NewResendAdapter(options ResendOptions) *ResendAdapter {
	return &ResendAdapter{
		email.BaseAdapter{
			Key: "resend",
			Credentials: map[string]string{
				"api_key": options.ApiKey,
			},
		},
	}
}

func (this *ResendAdapter) Clone() email.IEmailAdapter {
	return &ResendAdapter{
		email.BaseAdapter{
			Key:         this.Key,
			Credentials: this.CloneCredentials(),
		},
	}
}

func (this *ResendAdapter) Send(message email.EmailPayload, configuration ...email.EmailConfiguration) (string, error) {
	apiKey := this.Credentials["api_key"]
	if apiKey == "" {
		return "", errors.New("resend api key is missing")
	}

	client := resend.NewClient(apiKey)

	params := &resend.SendEmailRequest{
		From:    message.From,
		To:      message.To,
		Html:    message.Body,
		Subject: message.Subject,
	}

	if len(configuration) > 0 {
		params.Cc = configuration[0].Cc
		params.Bcc = configuration[0].Bcc
		params.ReplyTo = configuration[0].ReplyTo
		// Headers can be added if supported by resend-go
	}

	sent, err := client.Emails.Send(params)

	if err != nil {
		return "", err
	}

	if sent.Id == "" {
		return "", errors.New("email not sent")
	}

	return sent.Id, nil
}
