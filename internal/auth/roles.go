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
// Roles prefixed with ^ are excluded after all other roles are applied, so "all,^shell"
// grants everything except shell.
func ParseRole(input string) Role {
	var roles Role
	var excluded Role
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
		negated := strings.HasPrefix(role, "^")
		if negated {
			role = strings.TrimSpace(strings.TrimPrefix(role, "^"))
		}

		var bits Role
		switch role {
		case "shell", "dozzle_shell":
			bits = Shell
		case "actions", "dozzle_actions":
			bits = Actions
		case "download", "dozzle_download":
			bits = Download
		case "all", "dozzle_all":
			bits = All
		case "none", "dozzle_none":
			if negated {
				log.Debug().Str("role", role).Msg("none cannot be negated")
				continue
			}
			return None
		default:
			log.Debug().Str("role", role).Msg("invalid role")
			continue
		}

		if negated {
			excluded |= bits
		} else {
			roles |= bits
		}
	}

	return roles &^ excluded
}

func (roles Role) Has(role Role) bool {
	return roles&role != 0
}
