// Package config implements a minimal TOML parser sufficient for HOSA configuration.
// Supports: sections [section], subsections [section.sub], string/int/float values,
// and line comments (#). No arrays, dates, or inline tables — not needed.
//
// Zero external dependencies — consistent with the HOSA design principle.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// table is a flat map of "section.key" → raw string value.
type table map[string]string

// parseTOML reads a TOML file and returns a flat key→value map.
// Keys are fully qualified: "detection.alpha_ewma", "motor.cgroup_path", etc.
func parseTOML(path string) (table, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("parseTOML: cannot read %q: %w", path, err)
	}
	return parseTOMLBytes(data)
}

func parseTOMLBytes(data []byte) (table, error) {
	t := make(table)
	section := ""

	for lineno, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Section header: [detection] or [detection.alpha_per_probe]
		if strings.HasPrefix(line, "[") {
			end := strings.Index(line, "]")
			if end < 0 {
				return nil, fmt.Errorf("parseTOML: line %d: unclosed section header", lineno+1)
			}
			section = strings.TrimSpace(line[1:end])
			continue
		}

		// Key = value (strip inline comment)
		eq := strings.Index(line, "=")
		if eq < 0 {
			return nil, fmt.Errorf("parseTOML: line %d: expected key=value, got %q", lineno+1, line)
		}

		key := strings.TrimSpace(line[:eq])
		rawVal := strings.TrimSpace(line[eq+1:])

		// Strip inline comment
		if idx := strings.Index(rawVal, " #"); idx >= 0 {
			rawVal = strings.TrimSpace(rawVal[:idx])
		}

		// Strip quotes from string values
		if len(rawVal) >= 2 && rawVal[0] == '"' && rawVal[len(rawVal)-1] == '"' {
			rawVal = rawVal[1 : len(rawVal)-1]
		}

		fullKey := key
		if section != "" {
			fullKey = section + "." + key
		}

		t[fullKey] = rawVal
	}

	return t, nil
}

// getString returns the string value for a key, or the default if not found.
func (t table) getString(key, def string) string {
	if v, ok := t[key]; ok {
		return v
	}
	return def
}

// getFloat returns the float64 value for a key, or the default if not found or invalid.
func (t table) getFloat(key string, def float64) float64 {
	v, ok := t[key]
	if !ok {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

// getInt returns the int value for a key, or the default if not found or invalid.
func (t table) getInt(key string, def int) int {
	v, ok := t[key]
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}