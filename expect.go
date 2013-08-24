package expect

import (
	"errors"
	"github.com/kr/pty"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sync"
	"time"
)

type Expect struct {
	timeout time.Duration
	pty     *os.File
	buffer  []byte

	readChan   chan readEvent
	readStatus error
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
func Create(pty *os.File) (exp *Expect) {
	rv := Expect{}
	rv.pty = pty
	rv.timeout = time.Hour * 24 * 365
	rv.buffer = make([]byte, 0, 0)
	rv.readChan = make(chan readEvent)
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

// GetBuffer returns a copy of the contents of a buffer
func (exp *Expect) Buffer() string {
	exp.drainReadChan()
	return string(exp.buffer)
}

// Close will end the expect session and return any error associated with the close
func (exp *Expect) Close() error {
	// Remove any finalizer associated with exp
	runtime.SetFinalizer(exp, nil)
	return exp.pty.Close()
}

// Send data to program
func (exp *Expect) Send(s string) error {
	return exp.send([]byte(s), false)
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
	if exp.readStatus != nil {
		return Match{}, exp.readStatus
	}

	giveUpTime := time.Now().Add(exp.timeout)

	for first := true; first || time.Now().Before(giveUpTime); first = false {

		if !first {
			exp.drainOnceTimeout(giveUpTime)
		}

		// Check for a match or error
		if m, found := exp.checkForMatch(pat); found {
			return m, nil
		} else {
		}

		// No match, see if we have an error (Most common - io.EOF)
		if exp.readStatus != nil {
			return Match{}, exp.readStatus
		}
	}

	// Time is up.
	return Match{}, ErrTimeout
}

// Expect(s string) is equivalent to exp.ExpectRegexp(regexp.MustCompile(s))
func (exp *Expect) Expect(expr string) (m Match, err error) {
	return exp.ExpectRegexp(regexp.MustCompile(expr))
}

func (exp *Expect) ExpectEOF() error {
	_, err := exp.Expect("$a")
	return err
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
func (exp *Expect) drainReadChan() {
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
	// TODO observers
	for len(arr) > 0 {
		if n, err := exp.pty.Write(arr); err == nil {
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
		for !done {
			if len(queue) > 0 {
				select {
				case exp.readChan <- queue[0]:
					queue = queue[1:]
				case read, ok := <-queueInput:
					if ok {
						queue = append(queue, read)
					} else {
						done = true
					}
				}
			} else {
				read, ok := <-queueInput
				if ok {
					queue = append(queue, read)
				} else {
					done = true
				}
			}
		}

		// Drain queue
		for _,read := range queue {
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
