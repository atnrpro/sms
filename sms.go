package sms

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

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

type sender struct {
	login    string
	password string
	devMode  bool
}

// New is a constructor of sender.
func New(login, password string, opts ...Option) *sender {
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

type DeliveryStatus string

func (d DeliveryStatus) IsQueued() bool       { return string(d) == StatusQueued }
func (d DeliveryStatus) IsSent() bool         { return string(d) == StatusSent }
func (d DeliveryStatus) IsDelivered() bool    { return string(d) == StatusDelivered }
func (d DeliveryStatus) IsUnavailable() bool  { return string(d) == StatusUndeliveredUnavailable }
func (d DeliveryStatus) IsSpam() bool         { return string(d) == StatusUndeliveredSpam }
func (d DeliveryStatus) IsInvalidPhone() bool { return string(d) == StatusUndeliveredInvPhone }
func (d DeliveryStatus) IsUndelivered() bool {
	sd := string(d)
	return sd == StatusUndeliveredUnavailable || sd == StatusUndeliveredSpam || sd == StatusUndeliveredInvPhone
}
func (d DeliveryStatus) IsValid() bool {
	return d.IsQueued() || d.IsSent() || d.IsDelivered() || d.IsUndelivered()
}

// SendResult represents a result of sending an SMS.
type SendResult struct {
	SMSID     string
	SMSCnt    int
	SentAt    string
	DebugInfo string
}

// SendSMS sends an SMS right away with the default sender.
func (s *sender) SendSMS(to, text string) (SendResult, error) {
	return s.sendSMS(to, text, defaultFrom, "")
}

// SendSMSFrom sends an SMS right away from the specified sender.
func (s *sender) SendSMSFrom(to, text, from string) (SendResult, error) {
	return s.sendSMS(to, text, from, "")
}

// SendSMSAt sends an SMS from the default sender at the specified time.
func (s *sender) SendSMSAt(to, text, sendTime string) (SendResult, error) {
	return s.sendSMS(to, text, defaultFrom, sendTime)
}

// SendSMSFromAt sends an SMS from the specified sender at the specified time.
func (s *sender) SendSMSFromAt(to, text, from, sendTime string) (SendResult, error) {
	return s.sendSMS(to, text, from, sendTime)
}

func (s *sender) sendSMS(to, text, from, sendTime string) (SendResult, error) {
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
	respReader, err := s.request(URI+"/send", args)
	if err != nil {
		return SendResult{}, errors.New("failed to request the service: " + err.Error())
	}
	return s.parseSendSMSResponse(respReader)
}

func (s *sender) parseSendSMSResponse(resp io.Reader) (SendResult, error) {
	result := SendResult{}
	scanner := bufio.NewScanner(resp)
	// TODO: What if a scanner hits EOF?
	scanner.Scan() // FIXME: This line will be removed when sms-rassilka.com fixes an empty first line.
	scanner.Scan()
	code, err := strconv.Atoi(scanner.Text())
	if err != nil {
		return SendResult{}, errors.New("bad response code: " + err.Error())
	}
	if code < 0 {
		scanner.Scan()
		return SendResult{}, fmt.Errorf("error response: %d %s", code, scanner.Text())
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

func (s *sender) QueryStatus(SMSID string) (DeliveryStatus, error) {
	args := map[string]string{
		"smsId": SMSID,
	}
	respReader, err := s.request(URI+"/status", args)
	if err != nil {
		return "", errors.New("failed to request status: " + err.Error())
	}
	return s.parseStatusResponse(respReader)
}

func (s *sender) parseStatusResponse(resp io.Reader) (DeliveryStatus, error) {
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

func (s *sender) request(uri string, args map[string]string) (io.Reader, error) {
	// The error is caught during tests.
	req, _ := http.NewRequest(http.MethodGet, uri, nil)
	q := req.URL.Query()
	q.Set("login", s.login)
	q.Set("password", s.password)
	if s.devMode {
		q.Set("mode", "dev")
	}
	for k, v := range args {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	fmt.Println(req.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
