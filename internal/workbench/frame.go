package workbench

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

type wbMessage struct {
	Command string
	Parts   [][]byte
}

func encodeWB(command string, parts ...[]byte) []byte {
	var out strings.Builder
	_, _ = fmt.Fprintf(&out, "%d.%s", jsStringLen(command), command)
	for _, part := range parts {
		_, _ = fmt.Fprintf(&out, ",%d.%s", jsBytesStringLen(part), part)
	}
	out.WriteByte(';')
	return []byte(out.String())
}

func parseWB(data []byte) ([]wbMessage, error) {
	var messages []wbMessage
	for i := 0; i < len(data); {
		if data[i] == ';' {
			i++
			continue
		}

		command, next, err := parseSegment(data, i)
		if err != nil {
			return nil, err
		}
		msg := wbMessage{Command: string(command)}
		i = next

		for i < len(data) {
			switch data[i] {
			case ',':
				part, n, err := parseSegment(data, i+1)
				if err != nil {
					return nil, err
				}
				msg.Parts = append(msg.Parts, part)
				i = n
			case ';':
				i++
				messages = append(messages, msg)
				goto nextMessage
			default:
				return nil, fmt.Errorf("unexpected workbench frame byte %q at %d", data[i], i)
			}
		}
		return nil, errors.New("unterminated workbench message")

	nextMessage:
	}
	return messages, nil
}

func parseSegment(data []byte, start int) ([]byte, int, error) {
	i := start
	for i < len(data) && data[i] >= '0' && data[i] <= '9' {
		i++
	}
	if i == start || i >= len(data) || data[i] != '.' {
		return nil, 0, fmt.Errorf("invalid workbench segment at %d", start)
	}

	n, err := strconv.Atoi(string(data[start:i]))
	if err != nil {
		return nil, 0, err
	}
	i++
	// The protocol length is the JavaScript string length, not UTF-8 bytes.
	// Non-ASCII terminal data is common, so byte arithmetic cannot be used here.
	end, err := advanceJSStringLen(data, i, n)
	if err != nil {
		return nil, 0, fmt.Errorf("workbench segment length %d exceeds data at %d: %w", n, start, err)
	}
	return data[i:end], end, nil
}

func jsStringLen(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func jsBytesStringLen(b []byte) int {
	return len(utf16.Encode([]rune(string(b))))
}

func advanceJSStringLen(data []byte, start, units int) (int, error) {
	i := start
	for units > 0 {
		if i >= len(data) {
			return 0, errors.New("unexpected end of data")
		}
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 0 {
			return 0, errors.New("invalid utf-8")
		}
		unitCount := 1
		if r > 0xffff {
			unitCount = 2
		}
		if unitCount > units {
			return 0, errors.New("segment ends inside surrogate pair")
		}
		units -= unitCount
		i += size
	}
	return i, nil
}
