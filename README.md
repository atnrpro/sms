# sms-rassilka.com client library

[![Go Report Card](https://goreportcard.com/badge/github.com/tiabc/sms)](https://goreportcard.com/report/github.com/tiabc/sms)
[![Build Status](https://travis-ci.org/tiabc/sms.svg?branch=master)](https://travis-ci.org/tiabc/sms)
[![Coverage Status](https://coveralls.io/repos/github/tiabc/sms/badge.svg)](https://coveralls.io/github/tiabc/sms)

The API documentation can be found at https://sms-rassilka.com/downloads/api/infocity-http-get.pdf.

## Example

```go
package main

import (
	"fmt"
	"github.com/tiabc/sms"
)

func main() {
	// Create a sender with your login and MD5-hashed password.
	s := sms.NewSender("+7 999 888-77-66", "fd494182a7ee16ae07f641c7c03663d8")

	// Use any of SendSMS methods to send an SMS.
	res, err := s.SendSMS("+7 999 000-00-00", "Hello world!")
	if err != nil {
		panic(err)
	}

	// Query the delivery status of the just sent SMS.
	ds, err := s.QueryStatus(res.SMSID)
	if err != nil {
		panic(err)
	}
	
	// Process the delivery status.
	switch {
	case ds.IsInProgress():
		fmt.Println("Queued.")
	case ds.IsDelivered():
		fmt.Println("Delivered.")
	case ds.IsUndelivered():
		fmt.Println("Undelivered.")
	}
}
```

## License

MIT
