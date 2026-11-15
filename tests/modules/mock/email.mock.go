package mock

import (
	"auth_service/shared/email"
)

// FakeEmailAdapter records what would have been sent. Clone returns itself, so
// the sender's copy and the test's are the same recorder.
type FakeEmailAdapter struct {
	Sent []email.EmailPayload
	Err  error
}

var _ email.IEmailAdapter = &FakeEmailAdapter{}

func (this *FakeEmailAdapter) Send(message email.EmailPayload, _ ...email.EmailConfiguration) (string, error) {
	if this.Err != nil {
		return "", this.Err
	}

	this.Sent = append(this.Sent, message)

	return "sent-id", nil
}

func (this *FakeEmailAdapter) GetKey() string                     { return "fake" }
func (this *FakeEmailAdapter) GetCredentials() map[string]string  { return nil }
func (this *FakeEmailAdapter) SetCredentials(_ map[string]string) {}
func (this *FakeEmailAdapter) Clone() email.IEmailAdapter         { return this }
