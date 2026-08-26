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
	kind    lineKind
	text    string // comment or raw text
	key     string
	value   string
	comment string // inline trailing comment on a kindEntry line, without the marker
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
			value, comment := splitInlineComment(strings.TrimSpace(trimmed[idx+1:]))
			cur.items = append(cur.items, entry{kind: kindEntry, key: key, value: value, comment: comment})

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

// splitInlineComment looks for a ';' or '#' outside of any quoted portion of
// value and, if found, treats everything from there on as a trailing
// comment. Quotes let a value contain those characters literally, e.g.
// path = "C:\later;dir".
func splitInlineComment(value string) (string, string) {
	var quote rune
	for i, r := range value {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == ';' || r == '#':
			return strings.TrimSpace(value[:i]), strings.TrimSpace(value[i+1:])
		}
	}
	return value, ""
}

// block is a run of items that moves as a unit when a section is sorted.
// Most blocks are a single item; a run of comment lines directly above a
// key, with no blank line in between, is treated as attached to that key
// and forms one block together with it.
type block struct {
	items    []entry
	sortable bool
	key      string
}

// sortEntries reorders the key/value lines in a section alphabetically by
// key, case-insensitively. Blank lines, raw lines, and comments that aren't
// immediately followed by a key stay in their original slot.
func sortEntries(s *section) {
	var blocks []block
	for i := 0; i < len(s.items); {
		it := s.items[i]
		if it.kind == kindEntry {
			blocks = append(blocks, block{items: []entry{it}, sortable: true, key: it.key})
			i++
			continue
		}
		if it.kind == kindComment {
			j := i
			for j < len(s.items) && s.items[j].kind == kindComment {
				j++
			}
			if j < len(s.items) && s.items[j].kind == kindEntry {
				items := append([]entry(nil), s.items[i:j+1]...)
				blocks = append(blocks, block{items: items, sortable: true, key: s.items[j].key})
				i = j + 1
				continue
			}
		}
		blocks = append(blocks, block{items: []entry{it}})
		i++
	}

	var sortable []block
	for _, b := range blocks {
		if b.sortable {
			sortable = append(sortable, b)
		}
	}
	sort.SliceStable(sortable, func(i, j int) bool {
		return strings.ToLower(sortable[i].key) < strings.ToLower(sortable[j].key)
	})

	items := make([]entry, 0, len(s.items))
	n := 0
	for _, b := range blocks {
		if b.sortable {
			items = append(items, sortable[n].items...)
			n++
		} else {
			items = append(items, b.items...)
		}
	}
	s.items = items
}

// normalizeValue strips quotes around a value when they aren't doing any
// work, and prefers double quotes over single when quotes are kept. Quotes
// are considered load-bearing if the quoted content is empty, has leading
// or trailing whitespace, or contains a comment marker character (both of
// which would otherwise be lost or misread once unquoted). A single-quoted
// value is only switched to double quotes if it doesn't itself contain a
// double quote, so the swap can't change what the value means.
func normalizeValue(value string) string {
	if len(value) < 2 {
		return value
	}
	quote := rune(value[0])
	if (quote != '"' && quote != '\'') || rune(value[len(value)-1]) != quote {
		return value
	}
	inner := value[1 : len(value)-1]

	needsQuotes := inner == "" || inner != strings.TrimSpace(inner) || strings.ContainsAny(inner, ";#")
	if !needsQuotes {
		return inner
	}
	if quote == '\'' && !strings.Contains(inner, `"`) {
		return `"` + inner + `"`
	}
	return value
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
			line := it.key + " = " + normalizeValue(it.value)
			if it.comment != "" {
				line += " ; " + it.comment
			}
			lines = append(lines, line)
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
// "key = value" spacing style, one comment marker, quotes around values
// stripped unless they're load-bearing, at most one blank line between
// entries, and exactly one blank line before each section header. If
// sortKeys is true, entries within each section are sorted alphabetically
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
