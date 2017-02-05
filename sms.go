package sms

const (
	uri         = "https://sms-rassilka.com/api/simple"
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

// Sender is an object for sending SMS.
type Sender struct {
	Login    string
	Password string
	DevMode  bool
}

// DeliveryStatus represents a delivery status. If you need an exact status, compare with constants above.
type DeliveryStatus string

// IsInProgress tells if a message is still being processed.
func (d DeliveryStatus) IsInProgress() bool {
	return d == StatusQueued || d == StatusSent
}

// IsDelivered tells if a message has in fact been delivered.
func (d DeliveryStatus) IsDelivered() bool {
	return d == StatusDelivered
}

// IsUndelivered tells if a message has been processed and undelivered by any reason.
func (d DeliveryStatus) IsUndelivered() bool {
	return !d.IsInProgress() && !d.IsDelivered()
}
