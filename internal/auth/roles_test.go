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

		// Single dozzle_ prefixed role tests
		{"Single dozzle_shell role", "dozzle_shell", Shell},
		{"Single dozzle_actions role", "dozzle_actions", Actions},
		{"Single dozzle_download role", "dozzle_download", Download},
		{"Dozzle_none role", "dozzle_none", None},
		{"Dozzle_all role", "dozzle_all", All},

		// Case insensitive tests
		{"Shell uppercase", "SHELL", Shell},
		{"Actions mixed case", "AcTiOnS", Actions},
		{"Download with spaces", " download ", Download},
		{"Dozzle_shell uppercase", "DOZZLE_SHELL", Shell},
		{"Dozzle_actions mixed case", "DoZzLe_AcTiOnS", Actions},
		{"Dozzle_download with spaces", " dozzle_download ", Download},

		// Multiple roles with comma separator
		{"Shell and actions", "shell,actions", Shell | Actions},
		{"All three roles", "shell,actions,download", Shell | Actions | Download},
		{"Roles with spaces", "shell , actions , download", Shell | Actions | Download},
		{"Dozzle roles with comma", "dozzle_shell,dozzle_actions", Shell | Actions},
		{"Mixed dozzle and regular", "shell,dozzle_actions,download", Shell | Actions | Download},

		// Multiple roles with pipe separator
		{"Shell and actions with pipe", "shell|actions", Shell | Actions},
		{"All three with pipe", "shell|actions|download", Shell | Actions | Download},
		{"Mixed separators", "shell,actions|download", Shell | Actions | Download},
		{"Dozzle roles with pipe", "dozzle_shell|dozzle_actions", Shell | Actions},

		// JSON format tests
		{"JSON single role", `["shell"]`, Shell},
		{"JSON multiple roles", `["shell", "actions"]`, Shell | Actions},
		{"JSON all roles", `["shell", "actions", "download"]`, Shell | Actions | Download},
		{"JSON with spaces", ` ["shell", "actions"] `, Shell | Actions},
		{"JSON single dozzle role", `["dozzle_shell"]`, Shell},
		{"JSON multiple dozzle roles", `["dozzle_shell", "dozzle_actions"]`, Shell | Actions},
		{"JSON mixed dozzle and regular", `["shell", "dozzle_actions"]`, Shell | Actions},

		// Edge cases
		{"Empty string", "", None},
		{"Whitespace only", "   ", None},
		{"Invalid role", "invalid", None},
		{"Mixed valid and invalid", "shell,invalid,actions", Shell | Actions},
		{"None overrides others", "shell,none,actions", None},
		{"All overrides others", "shell,all,actions", All},
		{"Dozzle_none overrides others", "dozzle_shell,dozzle_none,dozzle_actions", None},
		{"Dozzle_all overrides others", "dozzle_shell,dozzle_all,dozzle_actions", All},

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
