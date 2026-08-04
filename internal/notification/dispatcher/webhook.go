package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

// errBlockedAddress is returned when a webhook URL resolves to a blocked
// address range. Loopback and link-local addresses are refused to prevent SSRF
// against the Dozzle host's own services and cloud metadata endpoints
// (e.g. 169.254.169.254). RFC1918 private ranges are intentionally allowed —
// self-hosted webhooks (Home Assistant, internal Mattermost, etc.) commonly
// live on private LANs.
var errBlockedAddress = errors.New("webhook target resolves to a blocked address range")

// zeroNetV4 covers 0.0.0.0/8 — on Linux these route to the local host.
var zeroNetV4 = &net.IPNet{IP: net.IP{0, 0, 0, 0}, Mask: net.CIDRMask(8, 32)}

func isBlockedIP(ip net.IP) bool {
	if isBlockedBaseIP(ip) {
		return true
	}
	// IPv6 transition mechanisms (6to4, NAT64, Teredo, IPv4-compatible) embed an
	// arbitrary IPv4 address that none of the checks above look at. Unwrap and
	// re-check the embedded address so 2002:7f00:1::1 is treated as 127.0.0.1.
	return slices.ContainsFunc(embeddedIPv4(ip), isBlockedBaseIP)
}

func isBlockedBaseIP(ip net.IP) bool {
	if ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsUnspecified() {
		return true
	}
	// 0.0.0.0/8: only 0.0.0.0 is caught by IsUnspecified; 0.x.x.x routes to
	// localhost on Linux and is abusable as an SSRF vector.
	if v4 := ip.To4(); v4 != nil && zeroNetV4.Contains(v4) {
		return true
	}
	// 255.255.255.255 limited broadcast
	if ip.Equal(net.IPv4bcast) {
		return true
	}
	return false
}

// embeddedIPv4 returns the IPv4 addresses carried inside an IPv6 transition
// address, or nil when the address carries none. Teredo yields two: the relay
// server and the (obfuscated) client.
func embeddedIPv4(ip net.IP) []net.IP {
	if ip.To4() != nil {
		return nil
	}
	ip16 := ip.To16()
	if ip16 == nil {
		return nil
	}

	switch {
	// 6to4 — RFC 3056, 2002::/16, IPv4 in bytes 2-6
	case ip16[0] == 0x20 && ip16[1] == 0x02:
		return []net.IP{net.IPv4(ip16[2], ip16[3], ip16[4], ip16[5])}

	// NAT64 well-known prefix — RFC 6052, 64:ff9b::/96, IPv4 in the low 32 bits
	case ip16[0] == 0x00 && ip16[1] == 0x64 && ip16[2] == 0xff && ip16[3] == 0x9b &&
		isZeros(ip16[4:12]):
		return []net.IP{net.IPv4(ip16[12], ip16[13], ip16[14], ip16[15])}

	// NAT64 local-use prefix — RFC 8215, 64:ff9b:1::/48. The embedded IPv4 sits
	// at a position that depends on the operator's prefix length, so block the
	// whole range rather than guess.
	case ip16[0] == 0x00 && ip16[1] == 0x64 && ip16[2] == 0xff && ip16[3] == 0x9b && ip16[4] == 0x00 && ip16[5] == 0x01:
		return []net.IP{net.IPv4zero}

	// Teredo — RFC 4380, 2001::/32. Server IPv4 in bytes 4-8, client IPv4 in
	// bytes 12-16 obfuscated by XOR with 0xff.
	case ip16[0] == 0x20 && ip16[1] == 0x01 && ip16[2] == 0x00 && ip16[3] == 0x00:
		return []net.IP{
			net.IPv4(ip16[4], ip16[5], ip16[6], ip16[7]),
			net.IPv4(ip16[12]^0xff, ip16[13]^0xff, ip16[14]^0xff, ip16[15]^0xff),
		}

	// IPv4-compatible — deprecated ::a.b.c.d, not unwrapped by net.IP.To4
	case isZeros(ip16[0:12]):
		return []net.IP{net.IPv4(ip16[12], ip16[13], ip16[14], ip16[15])}
	}

	return nil
}

func isZeros(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}

	var dialer net.Dialer
	var lastErr error
	for _, ip := range ips {
		if isBlockedIP(ip) {
			lastErr = errBlockedAddress
			continue
		}
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errBlockedAddress
	}
	return nil, lastErr
}

// UserAgent is set by the application at startup
var UserAgent = "Dozzle/head"

// WebhookDispatcher sends notifications to a webhook URL
type WebhookDispatcher struct {
	Name         string
	URL          string
	Template     *template.Template
	TemplateText string // Original template string for serialization
	Headers      map[string]string
	client       *http.Client
}

// NewWebhookDispatcher creates a new webhook dispatcher
// If templateStr is empty, the notification will be marshaled as JSON directly
func NewWebhookDispatcher(name, rawURL, templateStr string, headers map[string]string) (*WebhookDispatcher, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook URL: %w", err)
	}
	if scheme := strings.ToLower(parsed.Scheme); scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("invalid webhook URL scheme %q: only http and https are allowed", parsed.Scheme)
	}

	w := &WebhookDispatcher{
		Name:         name,
		URL:          rawURL,
		TemplateText: templateStr,
		Headers:      headers,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext:           safeDialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}

	if templateStr != "" {
		tmpl, err := template.New("webhook").Parse(templateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}
		w.Template = tmpl
	}

	return w, nil
}

// TestResult contains the result of a webhook test
type TestResult struct {
	Success    bool
	StatusCode int
	Error      string
}

// Send sends a notification to the webhook URL
func (w *WebhookDispatcher) Send(ctx context.Context, notification types.Notification) error {
	result := w.SendTest(ctx, notification)
	if !result.Success {
		return fmt.Errorf("webhook notification failed: %s", result.Error)
	}
	return nil
}

// SendTest sends a notification and returns detailed result for testing
func (w *WebhookDispatcher) SendTest(ctx context.Context, notification types.Notification) TestResult {
	var payload []byte
	var err error

	if w.Template != nil {
		payload, err = executeJSONTemplate(w.TemplateText, notification)
		if err != nil {
			return TestResult{Success: false, Error: fmt.Sprintf("failed to execute template: %v", err)}
		}
	} else {
		payload, err = json.Marshal(notification)
		if err != nil {
			return TestResult{Success: false, Error: fmt.Sprintf("failed to marshal notification: %v", err)}
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(payload))
	if err != nil {
		return TestResult{Success: false, Error: fmt.Sprintf("failed to create request: %v", err)}
	}

	for k, v := range w.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		if errors.Is(err, errBlockedAddress) {
			return TestResult{Success: false, Error: errBlockedAddress.Error()}
		}
		return TestResult{Success: false, Error: fmt.Sprintf("failed to send webhook: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Limit response body to 1MB; only used for operator-side debug logging,
		// never reflected back through the API response (would be an SSRF exfil sink).
		limitedReader := io.LimitReader(resp.Body, 1024*1024)
		responseBody, _ := io.ReadAll(limitedReader)
		log.Debug().
			Str("webhook", w.Name).
			Str("url", w.URL).
			Int("status_code", resp.StatusCode).
			Str("payload", string(payload)).
			Str("response_body", string(responseBody)).
			Msg("webhook returned non-success status code")
		return TestResult{
			Success:    false,
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("webhook returned status code %d", resp.StatusCode),
		}
	}

	return TestResult{Success: true, StatusCode: resp.StatusCode}
}

// executeJSONTemplate parses the template as JSON, resolves Go template placeholders
// in string values, and marshals back to JSON. This ensures all values are properly
// JSON-escaped regardless of their content (e.g., log messages containing quotes or braces).
func executeJSONTemplate(templateText string, data any) ([]byte, error) {
	var structure any
	if err := json.Unmarshal([]byte(templateText), &structure); err != nil {
		// Not valid JSON — fall back to raw text/template execution
		tmpl, parseErr := template.New("webhook").Parse(templateText)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse template: %w", parseErr)
		}
		var buf bytes.Buffer
		if execErr := tmpl.Execute(&buf, data); execErr != nil {
			return nil, fmt.Errorf("failed to execute template: %w", execErr)
		}
		return buf.Bytes(), nil
	}

	resolved, err := resolveTemplateValues(structure, data)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resolved)
}

// resolveTemplateValues recursively walks a JSON structure and executes
// Go template expressions found in string values.
func resolveTemplateValues(v any, data any) (any, error) {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, child := range val {
			resolved, err := resolveTemplateValues(child, data)
			if err != nil {
				return nil, err
			}
			result[k] = resolved
		}
		return result, nil
	case []any:
		result := make([]any, len(val))
		for i, child := range val {
			resolved, err := resolveTemplateValues(child, data)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil
	case string:
		if !strings.Contains(val, "{{") {
			return val, nil
		}
		tmpl, err := template.New("field").Parse(val)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template field %q: %w", val, err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to execute template field %q: %w", val, err)
		}
		return buf.String(), nil
	default:
		return val, nil
	}
}
