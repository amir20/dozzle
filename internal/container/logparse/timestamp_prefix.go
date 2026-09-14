package logparse

import "time"

const (
	// A zoned timestamp is a duplicate when it is this close to Docker's.
	// Loggers often truncate to the second, so the text can trail by up to 1s.
	timestampPrefixTolerance = 2 * time.Second
	// A timestamp without a zone is assumed to be local time, so it may be off
	// from Docker's UTC time by a whole number of quarter hours, up to this far.
	timestampPrefixMaxOffset = 14 * time.Hour
)

// timestampPrefixLen returns how many leading bytes of message are a timestamp
// that repeats dockerTs (unix milliseconds), or 0 when there is none.
//
// The prefix covers the timestamp's own ANSI coloring, an optional pair of
// brackets and the whitespace after it, so message[n:] starts at the first
// real character of the line. Everything it can match is ASCII, which means
// the byte length is also the UTF-16 length the frontend slices with.
//
// This runs on every plain log line, so it parses by hand and bails out on the
// first byte for lines that cannot start with a timestamp.
func timestampPrefixLen(message string, dockerTs int64) int {
	if dockerTs == 0 || message == "" {
		return 0
	}
	if c := message[0]; !isDigit(c) && c != '[' && c != 0x1b {
		return 0
	}

	i := 0
	colored := false
	var ok bool

	if i, ok = skipANSI(message, i, &colored); !ok {
		return 0
	}
	bracketed := i < len(message) && message[i] == '['
	if bracketed {
		i++
		if i, ok = skipANSI(message, i, &colored); !ok {
			return 0
		}
	}

	ts, zoned, end := parseTimestampPrefix(message, i)
	if end == 0 {
		return 0
	}
	i = end

	// Only resets may follow: the color that was set for the timestamp has to be
	// turned off before the text, or slicing it away would uncolor the rest.
	reset := false
	i = skipResets(message, i, &reset)
	if bracketed {
		if i >= len(message) || message[i] != ']' {
			return 0
		}
		i = skipResets(message, i+1, &reset)
	}
	if colored && !reset {
		return 0
	}

	spaces := 0
	for i < len(message) && (message[i] == ' ' || message[i] == '\t') {
		i++
		spaces++
	}
	if (!bracketed && spaces == 0) || i >= len(message) {
		return 0
	}

	if !matchesDockerTime(ts, zoned, dockerTs) {
		return 0
	}
	return i
}

func matchesDockerTime(ts int64, zoned bool, dockerTs int64) bool {
	diff := time.Duration(ts-dockerTs) * time.Millisecond
	if zoned {
		return absDuration(diff) <= timestampPrefixTolerance
	}
	if absDuration(diff) > timestampPrefixMaxOffset+timestampPrefixTolerance {
		return false
	}
	const quarter = 15 * time.Minute
	nearest := (diff + sign(diff)*quarter/2) / quarter * quarter
	return absDuration(diff-nearest) <= timestampPrefixTolerance
}

// skipANSI consumes SGR escape sequences (ESC [ ... m) starting at i. A
// non-SGR escape is not something we know how to drop safely, so it fails.
func skipANSI(s string, i int, colored *bool) (int, bool) {
	for i < len(s) && s[i] == 0x1b {
		end, isReset, ok := readSGR(s, i)
		if !ok {
			return i, false
		}
		if !isReset {
			*colored = true
		}
		i = end
	}
	return i, true
}

// skipResets consumes only SGR sequences that turn attributes off.
func skipResets(s string, i int, reset *bool) int {
	for i < len(s) && s[i] == 0x1b {
		end, isReset, ok := readSGR(s, i)
		if !ok || !isReset {
			return i
		}
		*reset = true
		i = end
	}
	return i
}

// readSGR reads one ESC [ params m sequence at i. isReset is true when every
// parameter only turns something off (0, 22, 23, 24, 39, 49 or empty).
func readSGR(s string, i int) (end int, isReset bool, ok bool) {
	if i+1 >= len(s) || s[i+1] != '[' {
		return i, false, false
	}
	isReset = true
	param := 0
	for j := i + 2; j < len(s) && j < i+32; j++ {
		c := s[j]
		switch {
		case isDigit(c):
			param = param*10 + int(c-'0')
		case c == ';' || c == 'm':
			switch param {
			case 0, 22, 23, 24, 39, 49:
			default:
				isReset = false
			}
			param = 0
			if c == 'm' {
				return j + 1, isReset, true
			}
		default:
			return i, false, false
		}
	}
	return i, false, false
}

// parseTimestampPrefix reads a date and time at s[i:] in one of these shapes:
//
//	2006-01-02T15:04:05Z
//	2006-01-02 15:04:05.000000+07:00
//	2006-01-02 15:04:05,000
//	2006/01/02 15:04:05
//
// It returns unix milliseconds (treating a zoneless time as UTC), whether a
// zone was present, and the index after the timestamp, or 0 when s[i:] is not
// a timestamp.
func parseTimestampPrefix(s string, i int) (ms int64, zoned bool, end int) {
	// yyyy-mm-dd hh:mm:ss is 19 bytes.
	if len(s)-i < 19 {
		return 0, false, 0
	}
	year, ok1 := digits(s, i, 4)
	sep := s[i+4]
	month, ok2 := digits(s, i+5, 2)
	day, ok3 := digits(s, i+8, 2)
	hour, ok4 := digits(s, i+11, 2)
	minute, ok5 := digits(s, i+14, 2)
	second, ok6 := digits(s, i+17, 2)
	if !(ok1 && ok2 && ok3 && ok4 && ok5 && ok6) ||
		(sep != '-' && sep != '/') || s[i+7] != sep ||
		(s[i+10] != 'T' && s[i+10] != ' ') ||
		s[i+13] != ':' || s[i+16] != ':' {
		return 0, false, 0
	}
	if month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59 || second > 60 {
		return 0, false, 0
	}

	j := i + 19
	nanos := 0
	if j < len(s) && (s[j] == '.' || s[j] == ',') {
		k := j + 1
		scale := 100_000_000
		for k < len(s) && isDigit(s[k]) {
			nanos += int(s[k]-'0') * scale
			scale /= 10
			k++
		}
		if k == j+1 {
			return 0, false, 0
		}
		j = k
	}

	offset := 0
	if j < len(s) {
		switch s[j] {
		case 'Z':
			zoned = true
			j++
		case '+', '-':
			// +07:00 or +0700
			if h, okH := digits(s, j+1, 2); okH {
				n := j + 3
				if n < len(s) && s[n] == ':' {
					n++
				}
				m, okM := digits(s, n, 2)
				if okM && h <= 14 && m < 60 {
					offset = h*3600 + m*60
					if s[j] == '-' {
						offset = -offset
					}
					zoned = true
					j = n + 2
				}
			}
		}
	}

	t := time.Date(year, time.Month(month), day, hour, minute, second, nanos, time.UTC)
	return t.UnixMilli() - int64(offset)*1000, zoned, j
}

func digits(s string, i, n int) (int, bool) {
	if i+n > len(s) {
		return 0, false
	}
	v := 0
	for k := i; k < i+n; k++ {
		if !isDigit(s[k]) {
			return 0, false
		}
		v = v*10 + int(s[k]-'0')
	}
	return v, true
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func sign(d time.Duration) time.Duration {
	if d < 0 {
		return -1
	}
	return 1
}
