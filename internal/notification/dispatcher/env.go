package dispatcher

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// envRef matches a ${VAR} placeholder. Bare $VAR is deliberately not expanded so
// a literal "$" in a URL or header value is left alone.
var envRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// blockedEnvPrefixes are variables a webhook may never read. Anyone who can edit
// a webhook chooses where it posts, so a placeholder is a way to send a variable
// to an arbitrary host. DOZZLE_ holds Dozzle's own auth and cloud settings; the
// rest are the usual cloud and engine credentials.
var blockedEnvPrefixes = []string{"DOZZLE_", "AWS_", "GOOGLE_", "AZURE_", "KUBERNETES_", "DOCKER_"}

// checkEnvRefs rejects any placeholder that names a blocked variable.
func checkEnvRefs(values ...string) error {
	for _, v := range values {
		for _, m := range envRef.FindAllStringSubmatch(v, -1) {
			name := strings.ToUpper(m[1])
			for _, prefix := range blockedEnvPrefixes {
				if strings.HasPrefix(name, prefix) {
					return fmt.Errorf("environment variable %s cannot be used in a webhook", m[1])
				}
			}
		}
	}
	return nil
}

// expandEnv replaces ${VAR} placeholders with values from this process's
// environment. An unset variable is an error rather than an empty string, so a
// typo fails loudly instead of posting to a truncated URL.
func expandEnv(s string) (string, error) {
	var missing string
	expanded := envRef.ReplaceAllStringFunc(s, func(ref string) string {
		name := envRef.FindStringSubmatch(ref)[1]
		v, ok := os.LookupEnv(name)
		if !ok && missing == "" {
			missing = name
		}
		return v
	})
	if missing != "" {
		return "", fmt.Errorf("environment variable %s is not set", missing)
	}
	return expanded, nil
}

// scrubEnv removes the values of every variable referenced in refs from msg.
// Errors from a failed send can quote the expanded URL in places redactURL does
// not reach (a DNS error names the host), and they are returned to whoever edits
// webhooks, so a placeholder must never come back as its value.
func scrubEnv(msg string, refs ...string) string {
	for _, ref := range refs {
		for _, m := range envRef.FindAllStringSubmatch(ref, -1) {
			if v := os.Getenv(m[1]); v != "" {
				msg = strings.ReplaceAll(msg, v, "[redacted]")
			}
		}
	}
	return msg
}

// redactURL keeps only scheme and host, since webhook URLs often carry the
// token in the path or query.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "[redacted]"
	}
	return u.Scheme + "://" + u.Host + "/[redacted]"
}
