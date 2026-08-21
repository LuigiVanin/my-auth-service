package email

import (
	"errors"
)

type EmailPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

type EmailConfiguration struct {
	Bcc     []string          `json:"bcc"`
	Cc      []string          `json:"cc"`
	ReplyTo string            `json:"reply_to"`
	Headers map[string]string `json:"headers"`
}

type IEmailAdapter interface {
	Send(message EmailPayload, configuration ...EmailConfiguration) (string, error)
	GetKey() string
	GetCredentials() map[string]string
	SetCredentials(credentials map[string]string)
	Clone() IEmailAdapter
}

type BaseAdapter struct {
	Key         string
	Credentials map[string]string
}

func (this *BaseAdapter) GetKey() string {
	return this.Key
}

func (this *BaseAdapter) GetCredentials() map[string]string {
	return this.Credentials
}

func (this *BaseAdapter) SetCredentials(credentials map[string]string) {
	this.Credentials = credentials
}

// Clone helper for deep copying credentials
func (this *BaseAdapter) CloneCredentials() map[string]string {
	if this.Credentials == nil {
		return nil
	}
	clone := make(map[string]string, len(this.Credentials))
	for k, v := range this.Credentials {
		clone[k] = v
	}
	return clone
}

type EmailSender struct {
	manager     *EmailManager
	adapter     IEmailAdapter
	credentials map[string]string
}

func NewEmailSender(manager *EmailManager) *EmailSender {
	return &EmailSender{
		manager: manager,
	}
}

func (this *EmailSender) UseAdapter(adapterKey string) *EmailSender {
	adapter := this.manager.GetAdapter(adapterKey)
	if adapter != nil {
		this.adapter = adapter.Clone()
	}
	return this
}

func (this *EmailSender) UseCredentials(credentials map[string]string) *EmailSender {
	this.credentials = credentials
	return this
}

func (this *EmailSender) Send(message EmailPayload, configuration ...EmailConfiguration) (string, error) {
	if this.adapter == nil {
		// Use default adapter if none selected
		defaultAdapter := this.manager.GetDefaultAdapter()
		if defaultAdapter == nil {
			return "", errors.New("no email adapters configured")
		}
		this.adapter = defaultAdapter.Clone()
	}

	if this.credentials != nil {
		this.adapter.SetCredentials(this.credentials)
	}

	return this.adapter.Send(message, configuration...)
}
