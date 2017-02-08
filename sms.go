package sms

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const (
	uri         = "https://sms-rassilka.com/api/simple"
	defaultFrom = "inform"

	// In progress delivery statuses.
	StatusQueued     = "0"
	StatusSent       = "1"
	StatusModerating = "10"

	// Successful delivery statuses.
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

	// Client allows to make requests with your own HTTP client.
	Client http.Client
}

// SendResult represents a result of sending an SMS.
type SendResult struct {
	SMSID     string
	SMSCnt    int
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
	args := map[string]string{
		"smsId": SMSID,
	}
	resp, err := s.request(uri+"/status", args)
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
	if code != "1" {
		return "", fmt.Errorf("error response: %s %s", code, t)
	}
	return DeliveryStatus(t), nil
}

func (s Sender) sendSMS(to, text, from, sendTime string) (SendResult, error) {
	args := map[string]string{
		"to":   to,
		"text": text,
	}
	if from != "" {
		args["from"] = from
	}
	if sendTime != "" {
		args["sendTime"] = sendTime
	}
	resp, err := s.request(uri+"/send", args)
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to request the service: %v", err)
	}
	defer resp.Close()
	return s.parseSendSMSResponse(resp)
}

func (s Sender) parseSendSMSResponse(resp io.Reader) (SendResult, error) {
	scanner := bufio.NewScanner(resp)
	scanner.Scan()
	code := scanner.Text()
	if code != "1" {
		scanner.Scan()
		return SendResult{}, fmt.Errorf("got error response: %s %s", code, scanner.Text())
	}

	sr := SendResult{}
	for line := 0; scanner.Scan(); line++ {
		switch line {
		case 0:
			sr.SMSID = scanner.Text()
		case 1:
			c, err := strconv.Atoi(scanner.Text())
			if err != nil {
				return SendResult{}, fmt.Errorf("bad SMS count: %v", err)
			}
			sr.SMSCnt = c
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

func (s Sender) request(uri string, args map[string]string) (io.ReadCloser, error) {
	// The error is caught during tests.
	req, _ := http.NewRequest(http.MethodGet, uri, nil)
	q := req.URL.Query()
	q.Set("login", s.Login)
	q.Set("password", s.PasswordMD5)
	if s.SandboxMode {
		q.Set("mode", "dev")
	}
	for k, v := range args {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	resp, err := s.Client.Do(req)
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
