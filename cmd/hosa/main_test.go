package main

import (
	"testing"
)

func TestReadMemTotal(t *testing.T) {
	bytes, err := readMemTotal()
	if err != nil {
		t.Fatalf("readMemTotal falhou: %v", err)
	}
	// Qualquer máquina real tem pelo menos 256 MB
	if bytes < 256*1024*1024 {
		t.Errorf("memTotal suspeito: %d bytes — menor que 256 MB", bytes)
	}
	t.Logf("memTotal = %d bytes (%.1f GB)", bytes, float64(bytes)/(1<<30))
}

func TestSplitLines(t *testing.T) {
	cases := []struct {
		input    string
		expected int
	}{
		{"a\nb\nc", 3},
		{"single", 1},
		{"a\n", 1},
		{"", 0},
	}
	for _, c := range cases {
		got := splitLines(c.input)
		if len(got) != c.expected {
			t.Errorf("splitLines(%q): esperado %d linhas, obtido %d", c.input, c.expected, len(got))
		}
	}
}

func TestSplitFields(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{"  16384000 kB", []string{"16384000", "kB"}},
		{"   42   ", []string{"42"}},
		{"", []string{}},
	}
	for _, c := range cases {
		got := splitFields(c.input)
		if len(got) != len(c.expected) {
			t.Errorf("splitFields(%q): esperado %v, obtido %v", c.input, c.expected, got)
			continue
		}
		for i := range c.expected {
			if got[i] != c.expected[i] {
				t.Errorf("splitFields(%q)[%d]: esperado %q, obtido %q", c.input, i, c.expected[i], got[i])
			}
		}
	}
}

func TestParseUint(t *testing.T) {
	cases := []struct {
		input    string
		expected uint64
	}{
		{"16384000", 16384000},
		{"0", 0},
		{"123abc", 123},
		{"", 0},
	}
	for _, c := range cases {
		got := parseUint(c.input)
		if got != c.expected {
			t.Errorf("parseUint(%q): esperado %d, obtido %d", c.input, c.expected, got)
		}
	}
}