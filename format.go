package main

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
)

type lineKind int

const (
	kindBlank lineKind = iota
	kindComment
	kindRaw
	kindEntry
)

type entry struct {
	kind  lineKind
	text  string // comment or raw text
	key   string
	value string
}

type section struct {
	name  string
	items []entry
}

// parse reads INI-ish text into an ordered list of sections. The first
// section always holds any keys that appear before the first [header] line.
func parse(r io.Reader) ([]*section, error) {
	global := &section{name: ""}
	sections := []*section{global}
	cur := global

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		switch {
		case trimmed == "":
			cur.items = append(cur.items, entry{kind: kindBlank})

		case strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#"):
			text := strings.TrimSpace(trimmed[1:])
			cur.items = append(cur.items, entry{kind: kindComment, text: text})

		case strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"):
			name := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			cur = &section{name: name}
			sections = append(sections, cur)

		case strings.Contains(trimmed, "="):
			idx := strings.Index(trimmed, "=")
			key := strings.TrimSpace(trimmed[:idx])
			value := strings.TrimSpace(trimmed[idx+1:])
			cur.items = append(cur.items, entry{kind: kindEntry, key: key, value: value})

		default:
			// Not blank, not a comment, no '=' and no brackets: keep it
			// verbatim rather than guessing what it was supposed to be.
			cur.items = append(cur.items, entry{kind: kindRaw, text: trimmed})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return sections, nil
}

// sortEntries reorders the key/value lines in a section alphabetically by
// key, case-insensitively. Comments, blanks and raw lines stay in their
// original slot, so a comment written above a key may no longer sit next
// to that same key once the keys have moved.
func sortEntries(s *section) {
	var positions []int
	var entries []entry
	for i, it := range s.items {
		if it.kind == kindEntry {
			positions = append(positions, i)
			entries = append(entries, it)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].key) < strings.ToLower(entries[j].key)
	})
	for n, pos := range positions {
		s.items[pos] = entries[n]
	}
}

// renderItems turns a section's items into normalised lines, one per item.
func renderItems(items []entry) []string {
	lines := make([]string, 0, len(items))
	for _, it := range items {
		switch it.kind {
		case kindBlank:
			lines = append(lines, "")
		case kindComment:
			if it.text == "" {
				lines = append(lines, ";")
			} else {
				lines = append(lines, "; "+it.text)
			}
		case kindRaw:
			lines = append(lines, it.text)
		case kindEntry:
			lines = append(lines, it.key+" = "+it.value)
		}
	}
	return lines
}

// collapseBlanks trims leading/trailing blank lines and squashes any run of
// blank lines down to a single one.
func collapseBlanks(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if l == "" && len(out) > 0 && out[len(out)-1] == "" {
			continue
		}
		out = append(out, l)
	}
	for len(out) > 0 && out[0] == "" {
		out = out[1:]
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

// Format reads an INI file from r and returns a normalised version: a single
// "key = value" spacing style, one comment marker, at most one blank line
// between entries, and exactly one blank line before each section header.
// If sortKeys is true, entries within each section are sorted alphabetically
// by key.
func Format(r io.Reader, sortKeys bool) (string, error) {
	sections, err := parse(r)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}

	var b strings.Builder
	for _, s := range sections {
		if sortKeys {
			sortEntries(s)
		}
		lines := collapseBlanks(renderItems(s.items))
		if s.name != "" {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("[" + s.name + "]\n")
		}
		for _, l := range lines {
			b.WriteString(l)
			b.WriteString("\n")
		}
	}

	result := strings.TrimRight(b.String(), "\n")
	if result == "" {
		return "", nil
	}
	return result + "\n", nil
}
