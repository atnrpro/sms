package sms

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"strconv"
	"fmt"
)

const (
	URI = "https://sms-rassilka.com/api/simple"
	defaultFrom = "inform"
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

// DevMode is an option specifying development operation mode.
func DevMode(s *sender) {
	s.devMode = true
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
	for line := 0; scanner.Scan(); line++ {
		switch line {
		case 0: // TODO: This line will be removed by the gateway.
		case 1:
			code, err := strconv.Atoi(scanner.Text())
			if err != nil {
				return SendResult{}, errors.New("bad response code: " + err.Error())
			}
			if code < 0 {
				return SendResult{}, fmt.Errorf("bad response code: %d", code)
			}
		// TODO: Read the human text in case of the error.
		case 2:
			result.SMSID = scanner.Text()
		case 3:
			c, err := strconv.Atoi(scanner.Text())
			if err != nil {
				return SendResult{}, errors.New("bad SMS count: " + err.Error())
			}
			result.SMSCnt = c
		case 4:
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
