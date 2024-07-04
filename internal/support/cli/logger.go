package cli

import (
	log "github.com/sirupsen/logrus"
)

func ConfigureLogger(level string) {
	if l, err := log.ParseLevel(level); err == nil {
		log.SetLevel(l)
	} else {
		panic(err)
	}

	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: true,
	})
}
