package expect

import (
	"errors"
	"github.com/jamesharr/eventbus"
	"github.com/kr/pty"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type Expect struct {
	timeout time.Duration
	pty     io.ReadWriteCloser
	buffer  []byte

	readChan   chan readEvent
	readStatus error

	eventbus *eventbus.EventBus

	closed bool
}

type Match struct {
	Before string
	Groups []string
}

type readEvent struct {
	buf    []byte
	status error
}

var ErrTimeout = errors.New("Expect Timeout")

const READ_SIZE = 4094

// Create an Expect instance from a command.
// Effectively the same as Create(pty.Start(exec.Command(name, args...)))
func Spawn(name string, args ...string) (*Expect, error) {
	pty, err := pty.Start(exec.Command(name, args...))
	if err != nil {
		return nil, err
	}
	return Create(pty), nil
}

// Create an Expect instance from something that we can do read/writes off of.
func Create(pty io.ReadWriteCloser) (exp *Expect) {
	rv := Expect{}
	rv.pty = pty
	rv.timeout = time.Hour * 24 * 365
	rv.buffer = make([]byte, 0, 0)
	rv.readChan = make(chan readEvent)
	rv.eventbus = eventbus.CreateEventBus()
	go rv.startReader()
	return &rv
}

// Timeout() returns amount of time an Expect() call will wait for the output to appear.
func (exp *Expect) Timeout() time.Duration {
	return exp.timeout
}

// SetTimeout(Duration) sets the amount of time an Expect() call will wait for the output to appear.
func (exp *Expect) SetTimeout(d time.Duration) {
	exp.timeout = d
}

// Buffer returns a copy of the contents of a buffer
func (exp *Expect) Buffer() string {
	return string(exp.buffer)
}

// Close will end the expect session and return any error associated with the close
func (exp *Expect) Close() error {
	if exp.closed {
		return nil
	}

	// Remove any finalizer associated with exp
	runtime.SetFinalizer(exp, nil)

	// Close up shop
	exp.closed = true
	exp.eventbus.Close()
	return exp.pty.Close()
}

// Send data to program
func (exp *Expect) Send(s string) error {
	return exp.send([]byte(s), false)
}

// Send data, but mark it as masked to observers. Use this for passwords
func (exp *Expect) SendMasked(s string) error {
	return exp.send([]byte(s), true)
}

// Send several lines data (separated by \n) to the process
func (exp *Expect) SendLn(lines ...string) error {
	for _, l := range lines {
		if err := exp.Send(l + "\n"); err != nil {
			return err
		}
	}
	return nil
}

// ExpectRegexp searches the I/O read stream for a pattern within .Timeout()
func (exp *Expect) ExpectRegexp(pat *regexp.Regexp) (Match, error) {
	exp.eventbus.Emit(&ObsExpectCall{pat, exp.timeout})

	if exp.readStatus != nil {
		exp.eventbus.Emit(&ObsExpectReturn{Match{}, exp.readStatus})
		return Match{}, exp.readStatus
	}

	giveUpTime := time.Now().Add(exp.timeout)

	for first := true; first || time.Now().Before(giveUpTime); first = false {

		// Read some data
		if !first {
			exp.drainOnceTimeout(giveUpTime)
		}

		// Check for a match
		if m, found := exp.checkForMatch(pat); found {
			exp.eventbus.Emit(&ObsExpectReturn{m, nil})
			return m, nil
		}

		// No match, see if we have an error (Most common - io.EOF)
		if exp.readStatus != nil {
			exp.eventbus.Emit(&ObsExpectReturn{Match{}, exp.readStatus})
			return Match{}, exp.readStatus
		}
	}

	// Time is up.
	exp.eventbus.Emit(&ObsExpectReturn{Match{}, ErrTimeout})
	return Match{}, ErrTimeout
}

// Expect(s string) is equivalent to exp.ExpectRegexp(regexp.MustCompile(s))
func (exp *Expect) Expect(expr string) (m Match, err error) {
	return exp.ExpectRegexp(regexp.MustCompile(expr))
}

// Wait for EOF
func (exp *Expect) ExpectEOF() error {
	_, err := exp.Expect("$EOF")
	return err
}

// Add an observer to the expect process.
//
// Observers get a copy of various I/O and API events.
//
// Note: observation channel is not closed when this
func (exp *Expect) AddObserver(observer chan eventbus.Message) {
	exp.eventbus.Register(observer)
}

// Remove an observer from the the expect process.
//
// This also closes the observer
func (exp *Expect) RemoveObserver(observer chan eventbus.Message) {
	exp.eventbus.Unregister(observer)
	close(observer)
}

func (exp *Expect) checkForMatch(pat *regexp.Regexp) (m Match, found bool) {

	matches := pat.FindSubmatchIndex(exp.buffer)
	if matches != nil {
		found = true
		groupCount := len(matches) / 2
		m.Groups = make([]string, groupCount)

		for i := 0; i < groupCount; i++ {
			start := matches[2*i]
			end := matches[2*i+1]
			if start >= 0 && end >= 0 {
				m.Groups[i] = string(exp.buffer[start:end])
			}
		}
		m.Before = string(exp.buffer[0:matches[0]])
		exp.buffer = exp.buffer[matches[1]:]
	}
	return
}

// Remove all the read events out of the
func (exp *Expect) DrainReadChan() {
	done := false
	for !done {
		select {
		case read, ok := <-exp.readChan:
			// Got some data, merge it
			if ok {
				exp.mergeRead(read)
			}

		default:
			// Nothing available, just return
			done = true
		}
	}
}

func (exp *Expect) drainOnceTimeout(giveUpTime time.Time) {
	wait := giveUpTime.Sub(time.Now())
	select {
	case read, ok := <-exp.readChan:
		// Got some data, merge it once.
		if ok {
			exp.mergeRead(read)
		}

	case <-time.After(wait):
		// Timeout, return
	}
}

func (exp *Expect) mergeRead(read readEvent) {
	exp.buffer = append(exp.buffer, read.buf...)
	exp.readStatus = read.status
	exp.fixNewLines()

	// Observation events
	if len(read.buf) > 0 {
		exp.eventbus.Emit(&ObsRecv{read.buf})
	}

	if read.status == io.EOF {
		exp.eventbus.Emit(&ObsEOF{})
	}
}

var newLineRegexp *regexp.Regexp
var newLineOnce sync.Once

// fixNewLines will change various newlines combinations to \r\n
func (exp *Expect) fixNewLines() {
	newLineOnce.Do(func() { newLineRegexp = regexp.MustCompile("\r\n") })

	// This code could probably be optimized
	exp.buffer = newLineRegexp.ReplaceAllLiteral(exp.buffer, []byte("\n"))
}

func (exp *Expect) send(arr []byte, masked bool) error {
	for len(arr) > 0 {
		if n, err := exp.pty.Write(arr); err == nil {
			exp.eventbus.Emit(&ObsSend{arr[0:n], masked})
			arr = arr[n:]
		} else {
			return err
		}
	}
	return nil
}

func (exp *Expect) startReader() {
	queueInput := make(chan readEvent)

	// Dynamic buffer channel shim
	go func() {
		queue := make([]readEvent, 0)
		done := false

		// These are left as variables to handle the len(queue)=0 case.
		var sendItem readEvent
		var sendChan chan readEvent = nil

		for !done {

			// Set up queue & send operation, otherwise let select block on nil-channel.
			if len(queue) > 0 {
				sendItem = queue[0]
				sendChan = exp.readChan
			} else {
				sendChan = nil
			}

			// Wait for which ever I/O event happens first
			select {
			case sendChan <- sendItem:
				queue = queue[1:]
			case read, ok := <-queueInput:
				if ok {
					queue = append(queue, read)
				} else {
					done = true
				}
			}
		}

		// Drain queue
		for _, read := range queue {
			// TODO - this hangs if the user exp.Close()s with data left to read.
			// exp.Close() should signal to us that there's no more listeners left.
			exp.readChan <- read
		}
		queue = nil
	}()

	// Reader process
	go func() {
		done := false
		for !done {
			buf := make([]byte, READ_SIZE)
			n, err := exp.pty.Read(buf)
			buf = buf[0:n]

			// OSX: Closed FD returns io.EOF
			// Linux: Closed FD returns syscall.EIO, translate to io.EOF
			pathErr, ok := err.(*os.PathError)
			if ok && pathErr.Err == syscall.EIO {
				err = io.EOF
			}

			queueInput <- readEvent{buf, err}

			if err != nil {
				done = true
			}
		}
		close(queueInput)
	}()
}

// TODO -- register finalizer, do we even need this?
func (exp *Expect) finalize() {
}
