package expect_test

import (
	"github.com/bmizerany/assert"
	"github.com/jamesharr/expect"
	"io"
	"io/ioutil"
	"testing"
	"time"
)

func simpleEchoTest(t *testing.T, exp *expect.Expect) {
	exp.SetTimeout(1 * time.Second)

	// Send data
	exp.SendLn("Hello")

	// Receive the term echo
	m, err := exp.Expect("[Hh]ello")
	assert.Equal(t, nil, err)
	assert.Equal(t, expect.Match{
		Before: "",
		Groups: []string{"Hello"},
	}, m)

	// Receive the real echo
	m, err = exp.Expect("[Hh]ello")
	assert.Equal(t, nil, err)
	assert.Equal(t, expect.Match{
		Before: "\n",
		Groups: []string{"Hello"},
	}, m)

	// Wait for EOF
	err = exp.ExpectEOF()
	assert.Equal(t, io.EOF, err)
}

func TestMocker(t *testing.T) {
	t.Log("Set up Mocker()")
	mocker := expect.CreateMocker()

	t.Log("Set up real expect process")
	exp, err := expect.Spawn("sh", "-c", "read line; echo $line")
	assert.Equal(t, nil, err)

	t.Log("Enable recording on expect")
	exp.AddObserver(mocker.GetObservationChannel())

	t.Log("Running simpleEchoTest on real program")
	simpleEchoTest(t, exp)
	t.Log("Done")
	t.Log("")

	// Get mocker data
	tcl_script := mocker.GetTCLExpectScript()
	t.Log("Expect Script:\n", tcl_script)
	f, err := ioutil.TempFile("", "mocker_test_tmpscript")
	assert.Equal(t, nil, err)

	tcl_script_filename := f.Name()
	f.WriteString(tcl_script)
	f.Close()

	// Create expect
	t.Log("Running simpleEchoTest with Mocked TCLScript")
	exp, err = expect.Spawn("expect", "-f", tcl_script_filename)
	assert.Equal(t, nil, err)

	t.Log("Running test")
	simpleEchoTest(t, exp)
	t.Log("Done")
	t.Log("")
}
