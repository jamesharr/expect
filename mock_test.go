package expect_test

import (
	"github.com/bmizerany/assert"
	"github.com/jamesharr/expect"
	"io"
	"testing"
	"time"
)

func simpleEchoTest(t *testing.T, exp *expect.Expect) {
	exp.SetTimeout(1 * time.Second)

	// Send data
	exp.SendLn("Hello\n")

	// Receive it back
	m, err := exp.Expect("[Hh]ello")
	assert.Equal(t, nil, err)
	assert.Equal(t, expect.Match{
		Before: "",
		Groups: []string{"Hello"},
	}, m)

	// Wait for EOF
	err = exp.ExpectEOF()
	assert.Equal(t, io.EOF, err)
}

func TestMocker(t *testing.T) {
	t.Log("Set up mock_recorder")
	mock_recorder := expect.CreateMockRecorder()

	t.Log("Set up real expect process")
	exp, err := expect.Spawn("sh", "-c", "read line; echo $line")
	assert.Equal(t, nil, err)

	t.Log("Enable recording on expect")
	exp.AddObserver(mock_recorder.GetObservationChannel())

	t.Log("Running simpleEchoTest on real program")
	simpleEchoTest(t, exp)
	t.Log("Done")
	t.Log("")

	// Get mocker data
	mock_config := mock_recorder.GetMockData()

	// Re-setup expect with mocker
	t.Log("Set up mock_player")
	mock_player := expect.CreateMockPlayer(mock_config)
	mock_player.CheckWrites = true
	mock_player.TimingMultiplier = 0.1

	// Create expect
	t.Log("Set up expect process with fake mock player")
	exp = expect.Create(mock_player)

	t.Log("Running simpleEchoTest on mock_player")
	simpleEchoTest(t, exp)
	t.Log("Done")
	t.Log("")
}
