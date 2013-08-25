package expect

import "github.com/jamesharr/eventbus"

type MockRecorder struct {
}

func (recorder *MockRecorder) GetObservationChannel() chan eventbus.Message {
	return nil
}

func (recorder *MockRecorder) GetMockData() *MockConfig {
	return nil
}
