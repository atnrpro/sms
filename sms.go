package sms

const (
	URI         = "https://sms-rassilka.com/api/simple"
	defaultFrom = "inform"

	// Successful delivery statuses.
	StatusQueued    = "0"
	StatusSent      = "1"
	StatusDelivered = "3"

	// Unsuccessful delivery statuses.
	StatusUndeliveredUnavailable = "4"
	StatusUndeliveredSpam        = "15"
	StatusUndeliveredInvPhone    = "16"

	// TODO: Other delivery statuses.
)

// Sender is the library Facade.
type Sender interface {
	SendSMS(to, text string) (SendResult, error)
	SendSMSFrom(to, text, from string) (SendResult, error)
	SendSMSAt(to, text, sendTime string) (SendResult, error)
	SendSMSFromAt(to, text, from, sendTime string) (SendResult, error)
	QueryStatus(SMSID string) (DeliveryStatus, error)
}

type sender struct {
	login    string
	password string
	devMode  bool
}

// New is a constructor of Sender.
func New(login, password string, opts ...Option) Sender {
	s := sender{
		login:    login,
		password: password,
	}
	for i := range opts {
		opts[i](&s)
	}
	return &s
}

// Option is an optional argument for the sender constructor.
type Option func(*sender)

// DevMode is an Option specifying development operation mode.
func DevMode(s *sender) {
	s.devMode = true
}

// DeliveryStatus represents a delivery status. If you need an exact status, compare with constants above.
type DeliveryStatus string

// IsInProcess tells if a message is still being processed.
func (d DeliveryStatus) IsInProcess() bool {
	return string(d) == StatusQueued || string(d) == StatusSent
}

// IsDelivered tells if a message has in fact been delivered.
func (d DeliveryStatus) IsDelivered() bool {
	return string(d) == StatusDelivered
}

// IsUndelivered tells if a message has been processed and undelivered by any reason.
func (d DeliveryStatus) IsUndelivered() bool {
	sd := string(d)
	return sd == StatusUndeliveredUnavailable || sd == StatusUndeliveredSpam || sd == StatusUndeliveredInvPhone
}
