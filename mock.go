package expect

func CreateMockRecorder() *MockRecorder {
	return &MockRecorder{}
}

func CreateMockPlayer(config *MockConfig) *MockPlayer {
	return &MockPlayer{}
}
