package expect

import "io"

type MockPlayer struct {
	CheckWrites      bool
	TimingMultiplier float32
}

func (player *MockPlayer) Read(p []byte) (n int, err error) {
	// TODO
	return 0, io.EOF
}

func (player *MockPlayer) Write(p []byte) (n int, err error) {
	// TODO
	return len(p), nil
}

func (player *MockPlayer) Close() error {
	return nil
}
