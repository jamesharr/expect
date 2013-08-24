package expect_test

import (
	"github.com/jamesharr/expect"
	"github.com/kr/pty"
	"io"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

func assertSame(t *testing.T, a, b interface{}) {
	if a != b {
		t.Logf("%#v == %#v assert failed", a, b)
		t.Fail()
	}
}

func assertEq(t *testing.T, a, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Logf("%#v == %#v assert failed", a, b)
		t.Fail()
	}
}

func TestTimeout(t *testing.T) {
	// Start basic
	t.Log("Starting Command")
	pty, err := pty.Start(exec.Command("bash", "-c", "sleep 0.1; echo hello"))
	assertSame(t, err, nil)
	exp := expect.Create(pty)

	// This should timeout
	t.Log("Expect - should timeout")
	exp.SetTimeout(time.Millisecond)
	m, err := exp.Expect("[Hh]ello")
	t.Logf(" err=%#v", err)
	assertSame(t, err, expect.ErrTimeout)
	assertEq(t, m.Before, "")
	assertEq(t, m.Groups, []string(nil))

	// Try to get get the final text
	t.Log("Test - should finish immediately")
	t.Logf(" buffer[pre]:%#v", exp.Buffer())
	exp.SetTimeout(time.Second)
	m, err = exp.Expect("e(l+)o")
	t.Logf(" m=%#v, err=%#v", m, err)
	assertSame(t, err, nil)
	assertEq(t, m, expect.Match{
		Before: "h",
		Groups: []string{"ello", "ll"},
	})

	// Test assert
	t.Log("Test should return an EOF")
//	t.Logf(" Buffer: %#v", exp.Buffer())
	err = exp.ExpectEOF()
	t.Logf(" err=%#v", err)
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
	// Start cat
	exp, err := expect.Spawn("cat")
	assertSame(t, err, nil)
	exp.SetTimeout(time.Second)

	// Sending large amounts of text
	t.Log("Generating large amounts of text")
	text := make([]byte, 128)
	for i := range text {
		text[i] = '.'
	}
	text[len(text)-1] = '\n'

	t.Log("Writing large amounts of text")
	for i := 0; i < 1024; i++ {
//		t.Logf(" Writing %d bytes", i*len(text))
		err := exp.Send(string(text))
		if err != nil {
			t.Logf(" Send Error: %#v", err)
		}
	}
	exp.Send("\nDONE\n")

	t.Log("Expecting to see finish message")
	match, err := exp.Expect("DONE")
	t.Logf(" match.Groups=%#v", match.Groups)
	t.Logf(" err=%#v", err)
	assertSame(t, err, nil)

}
