package cli

import (
	"os"
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"
)

func ValidateEnvVars(types ...any) {
	expectedEnvs := make(map[string]bool)
	for _, t := range types {
		typ := reflect.TypeOf(t)

		for field := range typ.Fields() {
			for tag := range strings.SplitSeq(field.Tag.Get("arg"), ",") {
				if after, ok := strings.CutPrefix(tag, "env:"); ok {
					expectedEnvs[after] = true
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
