package cli

import (
	"os"
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"
)

func ValidateEnvVars(types ...interface{}) {
	expectedEnvs := make(map[string]bool)
	for _, t := range types {
		typ := reflect.TypeOf(t)

		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			for _, tag := range strings.Split(field.Tag.Get("arg"), ",") {
				if strings.HasPrefix(tag, "env:") {
					expectedEnvs[strings.TrimPrefix(tag, "env:")] = true
				}
			}
		}
	}

	for _, env := range os.Environ() {
		actual := strings.Split(env, "=")[0]
		if strings.HasPrefix(actual, "DOZZLE_") && !expectedEnvs[actual] {
			log.Warn().Str("env", actual).Msg("Unexpected environment variable")
		}
	}
}
