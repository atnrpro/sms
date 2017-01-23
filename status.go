package sms

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

// QueryStatus requests delivery status of an SMS.
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

func (s *sender) parseStatusResponse(resp io.ReadCloser) (DeliveryStatus, error) {
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
