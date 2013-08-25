package expect_test

import (
	"github.com/jamesharr/expect"
	"io"
	"testing"
)

func simpleEchoTest(t *testing.T, exp *expect.Expect) {
	// Send data
	exp.SendLn("Hello\n")

	// Receive it back
	m, err := exp.Expect("[Hh]ello\n")
	assertEq(t, err, nil)
	assertEq(t, m, expect.Match{
		Before: "",
		Groups: []string{"Hello"},
	})

	// Close
	exp.Close()
	err = exp.ExpectEOF()
	assertEq(t, err, io.EOF)
}

func TestMocker(t *testing.T) {
	// Set up mocker
	mock_recorder := expect.CreateMockRecorder()

	// Set up expect
	exp, err := expect.Spawn("cat")
	assertEq(t, err, nil)

	exp.AddObserver(mock_recorder)

	// FIRST RUN
	simpleEchoTest(t, exp)

	// Get mocker data
	mock_config := mock_recorder.GetMockData()

	// Re-setup expect with mocker
	mock_player := expect.CreateMockPlayer(mock_config)
	mock_player.CheckWrites = true
	mock_player.TimingMultiplier = 0.1

	// Create expect
	exp = expect.Create(mock_player)

	simpleEchoTest(t, exp)
}
