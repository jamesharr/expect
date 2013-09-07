package expect

import (
	"github.com/jamesharr/eventbus"
	"log"
	"os"
)

// Simple logger for events.
//
// This should probably be refactored once we figure out what a useful API is.
func LoggingObserver(filename string) chan eventbus.Message {
	ch := make(chan eventbus.Message)
	go observer(filename, ch)
	return ch
}

func observer(filename string, ch chan eventbus.Message) {

	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	logger := log.New(f, "[expect] ", log.LstdFlags)

	for {
		msg, ok := <-ch
		if !ok {
			logger.Println("***Observation channel closed***")
			break
		}

		switch observation := msg.(type) {
		case *ObsSend:
			if observation.Masked {
				logger.Print("{SEND ***MASKED***}")
			} else {
				logger.Printf("{SEND %#v}", string(observation.Data))
			}
		case *ObsRecv:
			logger.Print(string(observation.Data))
		case *ObsEOF:
			logger.Println("***EOF***")
		case *ObsExpectCall:
			logger.Printf("{Expect %v}", observation.Regexp)
		case *ObsExpectReturn:
			if observation.Error != nil {
				logger.Printf("{ExpectError %v}", observation.Error)
			} else {
				logger.Printf("{ExpectMatch %v}", observation.Match)
			}
		default:
			logger.Println("Unknown observation")
		}
	}
}
