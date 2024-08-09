package cli

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func ConfigureLogger(level string) {
	if level, err := zerolog.ParseLevel(level); err == nil {
		zerolog.SetGlobalLevel(level)
	} else {
		panic(err)
	}

	_, dev := os.LookupEnv("DEV")

	if dev {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}
