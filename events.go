package expect

import (
	"fmt"
	"regexp"
	"time"
)

// Data sent by the user
type ObsSend struct {
	Data   []byte
	Masked bool
}

// Data received from program and processed.
//
// Note: This reflects data as pulled into the Expect algorithm, not all network buffer is reflected here.
type ObsRecv struct {
	Data []byte
}

type ObsEOF struct {
}

type ObsExpectCall struct {
	Regexp  *regexp.Regexp
	Timeout time.Duration
}

type ObsExpectReturn struct {
	Match Match
	Error error
}

func (obs ObsSend) String() string {
	data := string(obs.Data)
	if obs.Masked {
		data = "***MASKED***"
	}
	return fmt.Sprintf(`ObsSend{%#v}`, data)
}

func (obs ObsRecv) String() string {
	return fmt.Sprintf(`ObsRecv{%#v}`, string(obs.Data))
}

func (obs ObsEOF) String() string {
	return fmt.Sprintf(`ObsEOF{}`)
}

func (obs ObsExpectCall) String() string {
	return fmt.Sprintf(`ObsExpectCall{%#v,%#v}`, obs.Regexp.String(), obs.Timeout)
}

func (obs ObsExpectReturn) String() string {
	return fmt.Sprintf(`ObsExpectReturn{%#v,%#v}`, obs.Match, obs.Error)
}
