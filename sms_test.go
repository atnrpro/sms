package sms

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"testing"
	"gopkg.in/jarcoal/httpmock.v1"
	"net/url"
)

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
		StatusModerating,
	}
	for _, s := range statuses {
		t.Run("check status "+string(s), func(t *testing.T) {
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
		t.Run("check status "+string(s), func(t *testing.T) {
			require.True(t, s.IsUndelivered())
			require.False(t, s.IsInProgress())
			require.False(t, s.IsDelivered())
		})
	}
}

func TestParseStatusResponse_Success(t *testing.T) {
	// Arrange.
	resp := bytes.NewBufferString("1\n3")
	s := Sender{}

	// Act.
	ds, err := s.parseStatusResponse(resp)

	// Assert.
	require.Nil(t, err)
	require.Equal(t, DeliveryStatus(StatusDelivered), ds)
}

func TestParseStatusResponse_BadCode(t *testing.T) {
	// Arrange.
	resp := bytes.NewBufferString("Unexpected response")
	s := Sender{}

	// Act.
	_, err := s.parseStatusResponse(resp)

	// Assert.
	require.NotNil(t, err)
}

func TestParseSendSMSResponse_Success(t *testing.T) {
	// Arrange.
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

	// Act.
	r, err := s.parseSendSMSResponse(req)

	// Assert.
	require.Nil(t, err)
	require.Equal(t, "123", r.SMSID)
	require.Equal(t, 1, r.SMSCnt)
	// TODO: time.Time.
	require.Equal(t, "2016-10-16 15:00:00", r.SentAt)
	require.Equal(t, "Array\n(\n [to] => 79998887766\n [text] => Земля, прощай!\n [from] => Komandor\n [sendTime] => 2016-10-16 15:00:00\n)\n", r.DebugInfo)
}

func TestParseSendSMSResponse_Error(t *testing.T) {
	// Arrange.
	req := bytes.NewBufferString(`-1
Invalid login/password`)
	s := Sender{}

	// Act.
	_, err := s.parseSendSMSResponse(req)

	// Assert.
	require.NotNil(t, err)
}

func TestParseSendSMSResponse_MalformedRequest(t *testing.T) {
	// Arrange.
	req := bytes.NewBufferString(`1
123
1
`)
	s := Sender{}

	// Act.
	_, err := s.parseSendSMSResponse(req)

	// Assert.
	require.NotNil(t, err)
}

type fakeClient struct {
	RespToWrite string

	req *http.Request
}

func (c *fakeClient) Do(req *http.Request) (*http.Response, error) {
	c.req = req

	return &http.Response{
		Body: ioutil.NopCloser(bytes.NewBufferString(c.RespToWrite)),
	}, nil
}

func TestRequest_Success(t *testing.T) {
	// Arrange.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &fakeClient{}
	httpmock.RegisterResponder(http.MethodGet, "https://sms-rassilka.com/api/simple/send", c.Do)
	s := Sender{
		Login:       "+79998887766",
		PasswordMD5: "fd494182a7ee16ae07f641c7c03663d8",
		SandboxMode: true,
	}

	// Act.
	s.request("/send", url.Values{"user-arg": []string{"user-val"}})

	// Assert.
	require.Equal(t, "https", c.req.URL.Scheme)
	require.Equal(t, "sms-rassilka.com", c.req.URL.Host)
	require.Equal(t, "/api/simple/send", c.req.URL.Path)
	require.Equal(t, "+79998887766", c.req.URL.Query().Get("login"))
	require.Equal(t, "fd494182a7ee16ae07f641c7c03663d8", c.req.URL.Query().Get("password"))
	require.Equal(t, "dev", c.req.URL.Query().Get("mode"))
	require.Equal(t, "user-val", c.req.URL.Query().Get("user-arg"))
}

func TestSender_sendSMS(t *testing.T) {
	// Arrange.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &fakeClient{}
	httpmock.RegisterResponder(http.MethodGet, "https://sms-rassilka.com/api/simple/send", c.Do)
	s := Sender{}

	// Act.
	_, _ = s.sendSMS("+79008007060", "Hello world", "inform", "2016-10-16 15:00:00")

	// Assert.
	require.Equal(t, "/api/simple/send", c.req.URL.Path)
	require.Equal(t, "+79008007060", c.req.URL.Query().Get("to"))
	require.Equal(t, "Hello world", c.req.URL.Query().Get("text"))
	require.Equal(t, "inform", c.req.URL.Query().Get("from"))
	require.Equal(t, "2016-10-16 15:00:00", c.req.URL.Query().Get("sendTime"))
}

func TestSender_QueryStatus(t *testing.T) {
	// Arrange.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &fakeClient{}
	httpmock.RegisterResponder(http.MethodGet, "https://sms-rassilka.com/api/simple/status", c.Do)
	s := Sender{}

	// Act.
	_, _ = s.QueryStatus("848918")

	// Assert.
	require.Equal(t, "/api/simple/status", c.req.URL.Path)
	require.Equal(t, "848918", c.req.URL.Query().Get("smsId"))
}

func TestSender_SendSMS(t *testing.T) {
	// Arrange.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &fakeClient{}
	httpmock.RegisterResponder(http.MethodGet, "https://sms-rassilka.com/api/simple/send", c.Do)
	s := Sender{}

	// Act.
	s.SendSMS("+79008007060", "Hello world")

	// Assert.
	require.Equal(t, "+79008007060", c.req.URL.Query().Get("to"))
	require.Equal(t, "Hello world", c.req.URL.Query().Get("text"))
	require.Equal(t, "inform", c.req.URL.Query().Get("from"))
	require.Equal(t, "", c.req.URL.Query().Get("sendTime"))

	// Act.
	s.SendSMSAt("+79008007060", "Hello world", "2016-10-16 15:00:00")

	// Assert.
	require.Equal(t, "+79008007060", c.req.URL.Query().Get("to"))
	require.Equal(t, "Hello world", c.req.URL.Query().Get("text"))
	require.Equal(t, "inform", c.req.URL.Query().Get("from"))
	require.Equal(t, "2016-10-16 15:00:00", c.req.URL.Query().Get("sendTime"))

	// Act.
	s.SendSMSFrom("+79008007060", "Hello world", "supercompany")

	// Assert.
	require.Equal(t, "+79008007060", c.req.URL.Query().Get("to"))
	require.Equal(t, "Hello world", c.req.URL.Query().Get("text"))
	require.Equal(t, "supercompany", c.req.URL.Query().Get("from"))
	require.Equal(t, "", c.req.URL.Query().Get("sendTime"))

	// Act.
	s.SendSMSFromAt("+79008007060", "Hello world", "supercompany", "2016-10-16 15:00:00")

	// Assert.
	require.Equal(t, "+79008007060", c.req.URL.Query().Get("to"))
	require.Equal(t, "Hello world", c.req.URL.Query().Get("text"))
	require.Equal(t, "supercompany", c.req.URL.Query().Get("from"))
	require.Equal(t, "2016-10-16 15:00:00", c.req.URL.Query().Get("sendTime"))
}
