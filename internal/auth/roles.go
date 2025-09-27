package auth

import (
	"encoding/json"
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

// ParseRole parses a comma-separated string of roles and returns the corresponding Role.
func ParseRole(input string) Role {
	var roles Role
	var parts []string

	// Check if input is valid JSON
	trimmed := strings.TrimSpace(input)
	if json.Valid([]byte(trimmed)) {
		var jsonRoles []string
		if err := json.Unmarshal([]byte(trimmed), &jsonRoles); err == nil {
			parts = jsonRoles
		} else {
			log.Warn().Str("input", input).Msg("failed to parse JSON roles")
			return None
		}
	} else {
		// Split by both commas and pipes
		parts = strings.FieldsFunc(input, func(c rune) bool {
			return c == ',' || c == '|'
		})
	}

	for _, r := range parts {
		role := strings.TrimSpace(strings.ToLower(r))
		switch role {
		case "shell", "dozzle_shell":
			roles |= Shell
		case "actions", "dozzle_actions":
			roles |= Actions
		case "download", "dozzle_download":
			roles |= Download
		case "none", "dozzle_none":
			return None
		case "all", "dozzle_all":
			return All
		default:
			log.Debug().Str("role", role).Msg("invalid role")
		}
	}
	return roles
}

func (roles Role) Has(role Role) bool {
	return roles&role != 0
}
