package expect

import (
	"fmt"
	"github.com/jamesharr/eventbus"
	"strings"
	"time"
)

func CreateMocker() *Mocker {
	rv := &Mocker{
		make(chan []mockOp),
		make(chan eventbus.Message),
		[]mockOp{},
	}
	go rv.run()
	return rv
}

type Mocker struct {
	resultChan   chan []mockOp
	observerChan chan eventbus.Message
	results      []mockOp
}

type mockOp interface {
	Dump() string
	TCLExpectStr() string
}

type mockSend string
type mockRecv string
type mockEOF struct{}
type mockWait float32

func (op mockSend) Dump() string { return fmt.Sprintf("SEND %#v\n", string(op)) }
func (op mockRecv) Dump() string { return fmt.Sprintf("RECV %#v\n", string(op)) }
func (op mockEOF) Dump() string  { return fmt.Sprintf("EOF\n") }
func (op mockWait) Dump() string { return fmt.Sprintf("WAIT %0.3f\n", float32(op)) }

func (op mockSend) TCLExpectStr() string { return fmt.Sprintf("expect_user -- %#v\n", string(op)) }
func (op mockRecv) TCLExpectStr() string { return fmt.Sprintf("send_user -- %#v\n", string(op)) }
func (op mockEOF) TCLExpectStr() string  { return fmt.Sprintf("# EOF\n") }
func (op mockWait) TCLExpectStr() string { return fmt.Sprintf("sleep %0.3f\n", float32(op)) }

func (mocker *Mocker) run() {
	var lastOp time.Time
	ops := make([]mockOp, 0)

	// Main loop
	done := false
	for !done {
		select {
		case mocker.resultChan <- ops:
			// Report our results
		case msg, ok := <-mocker.observerChan:
			if !ok {
				done = true
			} else {
				ops = appendToOps(ops, msg, &lastOp)
			}
		}
	}

	// Send observation one last time
	mocker.resultChan <- ops
	close(mocker.resultChan)
}

func appendToOps(ops []mockOp, msg eventbus.Message, lastOp *time.Time) []mockOp {

	var op mockOp
	switch msg := msg.(type) {
	case *ObsSend:
		op = mockSend(msg.Data)
	case *ObsRecv:
		op = mockRecv(msg.Data)
	case *ObsEOF:
		op = mockEOF{}
	}

	if op != nil {

		t := time.Now()
		if !lastOp.IsZero() {
			ops = append(ops, mockWait(t.Sub(*lastOp).Seconds()))
		}
		*lastOp = t

		ops = append(ops, op)
	}

	return ops
}

func (mocker *Mocker) GetObservationChannel() chan eventbus.Message {
	return mocker.observerChan
}

func (mocker *Mocker) getOps() []mockOp {
	ops, ok := <-mocker.resultChan
	if ok {
		mocker.results = ops
	}
	return mocker.results
}

func (mocker *Mocker) GetTCLExpectScript() string {
	ops := mocker.getOps()

	script := make([]string, 0, len(ops)+1)
	script = append(script, "#!/usr/bin/expect -f\n")
	for _, op := range ops {
		script = append(script, op.TCLExpectStr())
	}
	return strings.Join(script, "")
}

func (mocker *Mocker) DumpOps() string {
	ops := mocker.getOps()

	script := make([]string, 0, len(ops))
	for _, op := range ops {
		script = append(script, op.Dump())
	}
	return strings.Join(script, "")
}
