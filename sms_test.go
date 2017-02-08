package sms

import (
	"testing"
	"github.com/stretchr/testify/require"
	"bytes"
	"net/http"
)

func TestNewSender(t *testing.T) {
	s := NewSender("login", "passwordMD5")
	require.Equal(t, Sender{
		Login:       "login",
		PasswordMD5: "passwordMD5",
		SandboxMode: false,
		Client:      http.DefaultClient,
	}, s)
}

func TestDeliveryStatus_IsDelivered(t *testing.T) {
	s := DeliveryStatus(StatusDelivered)
	require.True(t, s.IsDelivered())
	require.False(t, s.IsInProgress())
	require.False(t, s.IsUndelivered())
}

func TestDeliveryStatus_IsInProgress(t *testing.T) {
	statuses := []DeliveryStatus{
		StatusQueued,
		StatusSent,
	}
	for _, s := range statuses {
		t.Run("check status " + string(s), func(t *testing.T) {
			require.True(t, s.IsInProgress())
			require.False(t, s.IsDelivered())
			require.False(t, s.IsUndelivered())
		})
	}
}

func TestDeliveryStatus_IsUndelivered(t *testing.T) {
	statuses := []DeliveryStatus{
		StatusUnavailable,
		StatusSpam,
		StatusInvPhone,
		"AbsolutelyUnknownStatus",
	}
	for _, s := range statuses {
		t.Run("check status " + string(s), func(t *testing.T) {
			require.True(t, s.IsUndelivered())
			require.False(t, s.IsInProgress())
			require.False(t, s.IsDelivered())
		})
	}
}

func TestParseStatusResponse_Success(t *testing.T) {
	req := bytes.NewBufferString("1\n3")
	s := Sender{}
	ds, err := s.parseStatusResponse(req)
	require.Nil(t, err)
	require.Equal(t, DeliveryStatus(StatusDelivered), ds)
}

func TestParseStatusResponse_BadCode(t *testing.T) {
	req := bytes.NewBufferString("Unexpected response")
	s := Sender{}
	_, err := s.parseStatusResponse(req)
	require.NotNil(t, err)
}

func TestParseSendSMSResponse_Success(t *testing.T) {
	req := bytes.NewBufferString(`1
123
1
2016-10-16 15:00:00
Array
(
 [to] => 79998887766
 [text] => Земля, прощай!
 [from] => Komandor
 [sendTime] => 2016-10-16 15:00:00
)`)
	s := Sender{}
	r, err := s.parseSendSMSResponse(req)
	require.Nil(t, err)
	require.Equal(t, "123", r.SMSID)
	require.Equal(t, 1, r.SMSCnt)
	// TODO: time.Time.
	require.Equal(t, "2016-10-16 15:00:00", r.SentAt)
	require.Equal(t, "Array\n(\n [to] => 79998887766\n [text] => Земля, прощай!\n [from] => Komandor\n [sendTime] => 2016-10-16 15:00:00\n)\n", r.DebugInfo)
}

func TestParseSendSMSResponse_Error(t *testing.T) {
	req := bytes.NewBufferString(`-1
Invalid login/password`)
	s := Sender{}
	_, err := s.parseSendSMSResponse(req)
	require.NotNil(t, err)
}

func TestParseSendSMSResponse_MalformedRequest(t *testing.T) {
	req := bytes.NewBufferString(`1
123
1
`)
	s := Sender{}
	_, err := s.parseSendSMSResponse(req)
	require.NotNil(t, err)
}
