package email

type EmailManager struct {
	adapters []IEmailAdapter
}

func NewEmailManager(adapters ...IEmailAdapter) *EmailManager {
	return &EmailManager{
		adapters: adapters,
	}
}

func (this *EmailManager) GetAdapter(key string) IEmailAdapter {
	for _, adapter := range this.adapters {
		if adapter.GetKey() == key {
			return adapter
		}
	}
	return nil
}

func (this *EmailManager) GetDefaultAdapter() IEmailAdapter {
	if len(this.adapters) > 0 {
		return this.adapters[0]
	}
	return nil
}

func (this *EmailManager) Sender() *EmailSender {
	return NewEmailSender(this)
}
