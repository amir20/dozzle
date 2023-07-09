package docker

import (
	"bufio"
	"io"
	"sync"

	"github.com/docker/docker/pkg/stdcopy"
	log "github.com/sirupsen/logrus"
)

func ReaderConvertor(reader io.Reader, tty bool) (chan *LogEvent, chan error) {
	events := make(chan *LogEvent)
	errors := make(chan error)

	if tty {
		go stream(reader, STDOUT, events, errors, &sync.WaitGroup{})
	} else {
		errReader, errWriter := io.Pipe()
		outReader, outWriter := io.Pipe()
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func(wg *sync.WaitGroup) {
			_, err := stdcopy.StdCopy(outWriter, errWriter, reader)
			errReader.Close()
			outReader.Close()
			wg.Wait()
			errors <- err
		}(&wg)

		go stream(outReader, STDOUT, events, errors, &wg)
		go stream(errReader, STDERR, events, errors, &wg)
	}
	return events, errors
}

func stream(reader io.Reader, streamType StdType, events chan *LogEvent, errors chan error, wg *sync.WaitGroup) {
	br := bufio.NewReader(reader)
	iterator := NewEventGenerator(br, streamType)
	for {
		logEvent, readerError := iterator.Next()
		if readerError != nil {
			if readerError != io.ErrClosedPipe {
				errors <- readerError
			}
			break
		}
		events <- logEvent
	}
	wg.Done()
	log.Tracef("streaming %s finished", streamType)
}
