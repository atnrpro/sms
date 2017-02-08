package sms

import (
	"testing"
	"github.com/stretchr/testify/require"
	"bytes"
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
		StatusUndeliveredUnavailable,
		StatusUndeliveredSpam,
		StatusUndeliveredInvPhone,
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
	require.Equal(t, DeliveryStatus(StatusDelivered), ds)
	require.Nil(t, err)
}

func TestParseStatusResponse_BadCode(t *testing.T) {
	req := bytes.NewBufferString("Unexpected response")
	s := Sender{}
	_, err := s.parseStatusResponse(req)
	require.NotNil(t, err)
}



/*
TODO: Write tests.

1
123
1
2016-10-16 15:00:00
Array
(
 [to] => 79998887766
 [text] => Земля, прощай!
 [from] => Komandor
 [sendTime] => 2016-10-16 15:00:00
)
*/
