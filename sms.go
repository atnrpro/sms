package sms

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	// In progress delivery statuses.
	StatusQueued     = "0"
	StatusSent       = "1"
	StatusModerating = "10"

	// Successful delivery status.
	StatusDelivered = "3"

	// Unsuccessful delivery statuses.
	StatusUnavailable    = "4"
	StatusRejected       = "11"
	StatusSpam           = "15"
	StatusInvPhone       = "16"
	StatusStopListGlobal = "20"
	StatusStopListLocal  = "21"
	StatusExpired        = "25"

	// Outdated statuses for backward compatibility.
	StatusOld2 = "2"
	StatusOld5 = "5"
	StatusOld6 = "6"
)

const (
	uri         = "https://sms-rassilka.com/api/simple"
	defaultFrom = "inform"
	successResp = "1"
)

// Sender is a library facade for sending SMS and retrieving delivery statuses.
type Sender struct {
	// Login on https://sms-rassilka.com
	Login string

	// MD5-hash of your password.
	PasswordMD5 string

	// SandboxMode is used to test the connection without actually wasting your balance.
	// If false, real SMS are sent and real delivery statuses are retrieved.
	// If true, no SMS are really sent and delivery statuses are fake.
	SandboxMode bool
}

// SendResult represents a result of sending an SMS.
type SendResult struct {
	SMSID     string
	SMSCost   string
	SentAt    string
	DebugInfo string
}

// SendSMS sends an SMS right away with the default Sender.
func (s Sender) SendSMS(to, text string) (SendResult, error) {
	return s.sendSMS(to, text, defaultFrom, "")
}

// SendSMSFrom sends an SMS right away from the specified Sender.
func (s Sender) SendSMSFrom(to, text, from string) (SendResult, error) {
	return s.sendSMS(to, text, from, "")
}

// SendSMSAt sends an SMS from the default Sender at the specified time.
func (s Sender) SendSMSAt(to, text, sendTime string) (SendResult, error) {
	return s.sendSMS(to, text, defaultFrom, sendTime)
}

// SendSMSFromAt sends an SMS from the specified Sender at the specified time.
func (s Sender) SendSMSFromAt(to, text, from, sendTime string) (SendResult, error) {
	return s.sendSMS(to, text, from, sendTime)
}

// QueryStatus requests delivery status of an SMS.
func (s Sender) QueryStatus(SMSID string) (DeliveryStatus, error) {
	vals := url.Values{}
	vals.Set("smsId", SMSID)
	resp, err := s.request("/status", vals)
	if err != nil {
		return "", fmt.Errorf("failed to request status: %v", err.Error())
	}
	defer resp.Close()
	return s.parseStatusResponse(resp)
}

func (s Sender) parseStatusResponse(resp io.Reader) (DeliveryStatus, error) {
	scanner := bufio.NewScanner(resp)
	scanner.Scan()
	code := scanner.Text()
	scanner.Scan()
	t := scanner.Text()
	if code != successResp {
		return "", fmt.Errorf("error response: %s %s", code, t)
	}
	return DeliveryStatus(t), nil
}

func (s Sender) sendSMS(to, text, from, sendTime string) (SendResult, error) {
	vals := url.Values{}
	vals.Set("to", to)
	vals.Set("text", text)
	if from != "" {
		vals.Set("from", from)
	}
	if sendTime != "" {
		vals.Set("sendTime", sendTime)
	}
	resp, err := s.request("/send", vals)
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to request the service: %v", err)
	}
	defer resp.Close()
	return s.parseSendSMSResponse(resp)
}

func (s Sender) parseSendSMSResponse(resp io.Reader) (SendResult, error) {
	scanner := bufio.NewScanner(resp)
	scanner.Scan()
	scanner.Scan()
	code := scanner.Text()
	if code != successResp {
		scanner.Scan()
		return SendResult{}, fmt.Errorf("error response: %s %s", code, scanner.Text())
	}

	sr := SendResult{}
	for line := 0; scanner.Scan(); line++ {
		switch line {
		case 0:
			sr.SMSID = scanner.Text()
		case 1:
			sr.SMSCost = scanner.Text()
		case 2:
			sr.SentAt = scanner.Text()
		default:
			sr.DebugInfo += scanner.Text() + "\n"
		}
	}
	if sr.SMSID == "" {
		return SendResult{}, fmt.Errorf("empty SMSID in the response")
	}
	if sr.SentAt == "" {
		return SendResult{}, fmt.Errorf("empty SentAt in the response")
	}
	if err := scanner.Err(); err != nil {
		return SendResult{}, fmt.Errorf("bad response: %v", err.Error())
	}
	return sr, nil
}

func (s Sender) request(path string, args url.Values) (io.ReadCloser, error) {
	// The error is caught during tests.
	req, _ := http.NewRequest(http.MethodGet, uri+path, nil)
	args.Set("login", s.Login)
	args.Set("password", s.PasswordMD5)
	if s.SandboxMode {
		args.Set("mode", "dev")
	}
	req.URL.RawQuery = args.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// DeliveryStatus represents a delivery status. If you need an exact status, compare with constants above.
type DeliveryStatus string

// IsInProgress tells if a message is still being processed.
func (d DeliveryStatus) IsInProgress() bool {
	return d == StatusQueued || d == StatusSent || d == StatusModerating
}

// IsDelivered tells if a message has in fact been delivered.
func (d DeliveryStatus) IsDelivered() bool {
	return d == StatusDelivered
}

// IsUndelivered tells if a message has been processed and undelivered by any reason.
func (d DeliveryStatus) IsUndelivered() bool {
	return !d.IsInProgress() && !d.IsDelivered()
}
