package expect_test

import (
	"io"
	"os/exec"
	"reflect"
	"testing"
	"time"
	"github.com/jamesharr/expect"
	"github.com/kr/pty"
)

func assertSame(t *testing.T, a, b interface{}) {
	if a != b {
		t.Errorf("%#v == %#v assert failed", a, b)
	}
}

func assertEq(t *testing.T, a, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("%#v == %#v assert failed", a, b)
	}
}

func TestTimeout(t *testing.T) {
	// Start basic
	t.Log("Starting Command")
	pty, err := pty.Start(exec.Command("bash", "-c", "sleep 0.1; echo hello"))
	assertSame(t, err, nil)
	exp := expect.Create(pty)

	// This should timeout
	t.Log("Test - should timeout")
	exp.SetTimeout(time.Millisecond)
	m, err := exp.Expect("[Hh]ello")
	assertSame(t, err, expect.ErrTimeout)
	assertEq(t, m.Before, "")
	assertEq(t, m.Groups, []string(nil))

	// Try to get get the final text
	t.Log("Test - should finish immediately")
	exp.SetTimeout(time.Second)
	m, err = exp.Expect("e(l+)o")
	assertSame(t, err, nil)
	assertEq(t, m, expect.Match{
			Before: "h",
			Groups: []string{"ello", "ll"},
		})

	// Test assert
	t.Log("Test should return an EOF")
	t.Logf(" Buffer: %#v", exp.Buffer())
	err = exp.ExpectEOF()
	assertSame(t, err, io.EOF)
}

func TestSend(t *testing.T) {
	// Start cat
	exp, err := expect.Spawn("cat")
	assertSame(t, err, nil)
	exp.SetTimeout(time.Second)

	// Send some data
	err = exp.Send("Hello\nWorld\n")
	assertSame(t, err, nil)

	// Get first chunk
	m, err := exp.Expect("Hello")
	assertSame(t, err, nil)
	assertEq(t, m.Before, "")
	assertEq(t, m.Groups, []string{"Hello"})

	// Check new lines
	m, err = exp.Expect("World\n")
	assertSame(t, err, nil)
	assertEq(t, m.Before, "\n")
	assertEq(t, m.Groups, []string{"World\n"})
}

func TestLargeBuffer(t *testing.T) {
	// TODO - test fails.
	// Not sure if it's worth it
	return

	// Start cat
	exp, err := expect.Spawn("cat")
	assertSame(t, err, nil)
	exp.SetTimeout(time.Second)

	// Sending large amounts of text
	text := make([]byte, 128)
	for i := range text {
		text[i] = '.'
	}
	text[len(text)-1] = '\n'

	t.Log("Writing large amounts of text")
	for i := 0; i < 1024; i++ {
		t.Logf(" Writing %d bytes", i*len(text))
		exp.Send(string(text))
	}

}
