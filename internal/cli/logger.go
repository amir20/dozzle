package cli

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func ConfigureLogger(level string) {
	if level, err := zerolog.ParseLevel(level); err == nil {
		zerolog.SetGlobalLevel(level)
		log.Logger = log.With().Str("version", Version).Logger()
	} else {
		panic(err)
	}

	_, dev := os.LookupEnv("DEV")

	if dev {
		writer := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.FieldsOrder = []string{"id", "from", "to", "since"}
		})
		log.Logger = log.Output(writer)
	}
}
