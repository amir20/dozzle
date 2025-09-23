package auth

import (
	"strings"

	"github.com/rs/zerolog/log"
)

type Role int

const (
	None  Role = 0
	Shell Role = 1 << iota
	Actions
	Download
)

const All = Shell | Actions | Download

// ParseRole parses a comma-separated string of roles and returns the corresponding Role. Default is All for empty input.
func ParseRole(commaValues string) Role {
	if strings.TrimSpace(commaValues) == "" {
		return All
	}

	var roles Role
	for r := range strings.SplitSeq(commaValues, ",") {
		role := strings.TrimSpace(strings.ToLower(r))
		switch role {
		case "shell":
			roles |= Shell
		case "actions":
			roles |= Actions
		case "download":
			roles |= Download
		case "none":
			return None
		case "all":
			return All
		default:
			log.Warn().Str("role", role).Msg("invalid role")
		}
	}
	return roles
}

func (roles Role) Has(role Role) bool {
	return roles&role != 0
}
