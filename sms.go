package sms

import (
	"io"
	"net/http"
	"bufio"
	"errors"
	"strconv"
	"fmt"
)

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

// SendResult represents a result of sending an SMS.
type SendResult struct {
	SMSID     string
	SMSCnt    int
	SentAt    string
	DebugInfo string
}

// SendSMS sends an SMS right away with the default Sender.
func (s *Sender) SendSMS(to, text string) (SendResult, error) {
	return s.sendSMS(to, text, defaultFrom, "")
}

// SendSMSFrom sends an SMS right away from the specified Sender.
func (s *Sender) SendSMSFrom(to, text, from string) (SendResult, error) {
	return s.sendSMS(to, text, from, "")
}

// SendSMSAt sends an SMS from the default Sender at the specified time.
func (s *Sender) SendSMSAt(to, text, sendTime string) (SendResult, error) {
	return s.sendSMS(to, text, defaultFrom, sendTime)
}

// SendSMSFromAt sends an SMS from the specified Sender at the specified time.
func (s *Sender) SendSMSFromAt(to, text, from, sendTime string) (SendResult, error) {
	return s.sendSMS(to, text, from, sendTime)
}

// QueryStatus requests delivery status of an SMS.
func (s *Sender) QueryStatus(SMSID string) (DeliveryStatus, error) {
	args := map[string]string{
		"smsId": SMSID,
	}
	respReader, err := s.request(uri+"/status", args)
	if err != nil {
		return "", errors.New("failed to request status: " + err.Error())
	}
	return s.parseStatusResponse(respReader)
}

func (s *Sender) parseStatusResponse(resp io.ReadCloser) (DeliveryStatus, error) {
	defer resp.Close()
	scanner := bufio.NewScanner(resp)
	// TODO: What if a scanner hits EOF?
	scanner.Scan() // FIXME: This line will be removed when sms-rassilka.com fixes an empty first line.
	scanner.Scan()
	code := scanner.Text()
	scanner.Scan()
	t := scanner.Text()
	if code != "1" {
		return "", fmt.Errorf("error response: %s %s", code, t)
	}
	return DeliveryStatus(t), nil
}

func (s *Sender) sendSMS(to, text, from, sendTime string) (SendResult, error) {
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
	respReader, err := s.request(uri+"/send", args)
	if err != nil {
		return SendResult{}, errors.New("failed to request the service: " + err.Error())
	}
	return s.parseSendSMSResponse(respReader)
}

func (s *Sender) parseSendSMSResponse(resp io.ReadCloser) (SendResult, error) {
	defer resp.Close()
	result := SendResult{}
	scanner := bufio.NewScanner(resp)
	// TODO: What if a scanner hits EOF?
	scanner.Scan() // FIXME: This line will be removed when sms-rassilka.com fixes an empty first line.
	scanner.Scan()
	code := scanner.Text()
	if code != "1" {
		scanner.Scan()
		return SendResult{}, errors.New("got error response: " + code + " " + scanner.Text())
	}

	for line := 0; scanner.Scan(); line++ {
		switch line {
		case 0:
			result.SMSID = scanner.Text()
		case 1:
			c, err := strconv.Atoi(scanner.Text())
			if err != nil {
				return SendResult{}, errors.New("bad SMS count: " + err.Error())
			}
			result.SMSCnt = c
		case 2:
			result.SentAt = scanner.Text()
		default:
			result.DebugInfo += scanner.Text() + "\n"
		}
	}
	if err := scanner.Err(); err != nil {
		return SendResult{}, errors.New("bad response" + err.Error())
	}
	return result, nil
}

func (s *Sender) request(uri string, args map[string]string) (io.ReadCloser, error) {
	// The error is caught during tests.
	req, _ := http.NewRequest(http.MethodGet, uri, nil)
	q := req.URL.Query()
	q.Set("login", s.Login)
	q.Set("password", s.Password)
	if s.DevMode {
		q.Set("mode", "dev")
	}
	for k, v := range args {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
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
