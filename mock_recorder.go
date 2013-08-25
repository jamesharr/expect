package expect

type MockRecorder struct {
}

func (recorder *MockRecorder) GetObservationChannel() chan interface{} {
	return nil
}

func (recorder *MockRecorder) GetMockData() *MockConfig {
	return nil
}
