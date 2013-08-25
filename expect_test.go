package expect_test

import (
	"github.com/bmizerany/assert"
	"github.com/jamesharr/expect"
	"github.com/kr/pty"
	"io"
	"os/exec"
	"testing"
	"time"
)

func TestExpect_timeout(t *testing.T) {
	// Start basic
	t.Log("Starting Command")
	pty, err := pty.Start(exec.Command("bash", "-c", "sleep 0.1; echo hello"))
	assert.Equal(t, nil, err)
	exp := expect.Create(pty)

	// This should timeout
	t.Log("Expect - should timeout")
	exp.SetTimeout(time.Millisecond)
	m, err := exp.Expect("[Hh]ello")
	t.Logf(" err=%#v", err)
	assert.Equal(t, expect.ErrTimeout, err)
	assert.Equal(t, "", m.Before)
	assert.Equal(t, []string(nil), m.Groups)

	// Try to get get the final text
	t.Log("Test - should finish immediately")
	t.Logf(" buffer[pre]:%#v", exp.Buffer())
	exp.SetTimeout(time.Second)
	m, err = exp.Expect("e(l+)o")
	t.Logf(" m=%#v, err=%#v", m, err)
	assert.Equal(t, nil, err)
	assert.Equal(t, expect.Match{
		Before: "h",
		Groups: []string{"ello", "ll"},
	}, m)

	// Test assert
	t.Log("Test should return an EOF")
	//	t.Logf(" Buffer: %#v", exp.Buffer())
	err = exp.ExpectEOF()
	t.Logf(" err=%#v", err)
	assert.Equal(t, io.EOF, err)
}

func TestExpect_send(t *testing.T) {
	// Start cat
	exp, err := expect.Spawn("cat")
	assert.Equal(t, nil, err)
	exp.SetTimeout(time.Second)

	// Send some data
	err = exp.Send("Hello\nWorld\n")
	assert.Equal(t, nil, err)

	// Get first chunk
	m, err := exp.Expect("Hello")
	assert.Equal(t, nil, err)
	assert.Equal(t, "", m.Before)
	assert.Equal(t, []string{"Hello"}, m.Groups)

	// Check new lines
	m, err = exp.Expect("World\n")
	assert.Equal(t, nil, err)
	assert.Equal(t, "\n", m.Before)
	assert.Equal(t, []string{"World\n"}, m.Groups)
}

func TestExpect_largeBuffer(t *testing.T) {
	// Start cat
	exp, err := expect.Spawn("cat")
	assert.Equal(t, nil, err)
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
	assert.Equal(t, nil, err)

}
