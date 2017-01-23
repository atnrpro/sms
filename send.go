package sms

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"strconv"
)

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

func (s *sender) parseSendSMSResponse(resp io.ReadCloser) (SendResult, error) {
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

func (s *sender) request(uri string, args map[string]string) (io.ReadCloser, error) {
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
