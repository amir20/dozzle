package auth

import (
	"testing"
)

func TestParseRole(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected Role
	}{
		// Single role tests
		{"Single shell role", "shell", Shell},
		{"Single actions role", "actions", Actions},
		{"Single download role", "download", Download},
		{"None role", "none", None},
		{"All role", "all", All},

		// Case insensitive tests
		{"Shell uppercase", "SHELL", Shell},
		{"Actions mixed case", "AcTiOnS", Actions},
		{"Download with spaces", " download ", Download},

		// Multiple roles with comma separator
		{"Shell and actions", "shell,actions", Shell | Actions},
		{"All three roles", "shell,actions,download", Shell | Actions | Download},
		{"Roles with spaces", "shell , actions , download", Shell | Actions | Download},

		// Multiple roles with pipe separator
		{"Shell and actions with pipe", "shell|actions", Shell | Actions},
		{"All three with pipe", "shell|actions|download", Shell | Actions | Download},
		{"Mixed separators", "shell,actions|download", Shell | Actions | Download},

		// JSON format tests
		{"JSON single role", `["shell"]`, Shell},
		{"JSON multiple roles", `["shell", "actions"]`, Shell | Actions},
		{"JSON all roles", `["shell", "actions", "download"]`, Shell | Actions | Download},
		{"JSON with spaces", ` ["shell", "actions"] `, Shell | Actions},

		// Edge cases
		{"Empty string", "", None},
		{"Whitespace only", "   ", None},
		{"Invalid role", "invalid", None},
		{"Mixed valid and invalid", "shell,invalid,actions", Shell | Actions},
		{"None overrides others", "shell,none,actions", None},
		{"All overrides others", "shell,all,actions", All},

		// Invalid JSON
		{"Invalid JSON format", `["shell"`, None},
		{"Malformed JSON", `{shell: "test"}`, None},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseRole(tc.input)
			if result != tc.expected {
				t.Errorf("ParseRole(%q) = %d, expected %d", tc.input, int(result), int(tc.expected))
			}
		})
	}
}
