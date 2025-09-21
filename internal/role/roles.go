package role

import (
	"strings"
)

type Role string

const (
	Shell    Role = "shell"
	Actions  Role = "actions"
	Download Role = "download"
)

var AllRoles = []Role{Shell, Actions, Download}

type UserRoles []Role

func (ur UserRoles) Exists() bool {
	return len(ur) > 0
}

func IsValidRole(role Role) bool {
	for _, r := range AllRoles {
		if role == r {
			return true
		}
	}
	return false
}

func ParseUserRole(commaValues string) UserRoles {
	if commaValues == "" {
		return UserRoles{}
	}

	roles := make(UserRoles, 0)
	seen := make(map[Role]struct{})

	for r := range strings.SplitSeq(commaValues, ",") {
		role := Role(strings.TrimSpace(r))
		if role == "" {
			continue
		}
		if !IsValidRole(role) {
			continue
		}

		if _, exists := seen[role]; !exists {
			seen[role] = struct{}{}
			roles = append(roles, role)
		}
	}
	return roles
}

func (ur UserRoles) HasRole(role Role) bool {
	for _, r := range ur {
		if r == role {
			return true
		}
	}
	return false
}
