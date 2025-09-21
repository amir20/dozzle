package auth

import (
	"strings"
)

type Role int

const (
	NONE  Role = 0
	Shell Role = 1 << iota
	Actions
	Download
)

const AllRole = Shell | Actions | Download

func ParseRole(commaValues string) Role {
	if commaValues == "" {
		return AllRole
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
			return NONE
		}
	}
	return roles
}

func (roles Role) Has(role Role) bool {
	return roles&role != 0
}
