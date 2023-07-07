package docker

import (
	"bufio"
	"io"

	"github.com/docker/docker/pkg/stdcopy"
	log "github.com/sirupsen/logrus"
)

func ReaderConvertor(reader io.Reader, tty bool) (chan *LogEvent, chan error) {
	events := make(chan *LogEvent, 100)
	errors := make(chan error)

	if tty {
		go stream(reader, STDOUT, events, errors)
	} else {
		errReader, errWriter := io.Pipe()
		outReader, outWriter := io.Pipe()
		go func() {
			_, err := stdcopy.StdCopy(outWriter, errWriter, reader)
			errors <- err
			errReader.Close()
			outReader.Close()
		}()

		go stream(outReader, STDOUT, events, errors)
		go stream(errReader, STDERR, events, errors)
	}
	return events, errors
}

func stream(reader io.Reader, streamType StdType, events chan *LogEvent, errors chan error) {
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
	log.Tracef("streaming %s finished", streamType)
}
