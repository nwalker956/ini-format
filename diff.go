package main

import (
	"fmt"
	"strings"
)

// op is one line of an edit script turning a into b: unchanged ('='),
// removed from a ('-'), or added in b ('+').
type op struct {
	tag  byte
	text string
}

// diffLines computes a minimal edit script from a to b using an LCS table.
// It's O(n*m) time and space, which is fine for config-file-sized input.
func diffLines(a, b []string) []op {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case a[i] == b[j]:
				dp[i][j] = dp[i+1][j+1] + 1
			case dp[i+1][j] >= dp[i][j+1]:
				dp[i][j] = dp[i+1][j]
			default:
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	var ops []op
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, op{tag: '=', text: a[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, op{tag: '-', text: a[i]})
			i++
		default:
			ops = append(ops, op{tag: '+', text: b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, op{tag: '-', text: a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, op{tag: '+', text: b[j]})
	}
	return ops
}

// lineCounts returns, for each index k in 0..len(ops), how many lines of a
// (or b) the first k ops account for. It lets a hunk's line-number range be
// read off directly instead of recounted for every hunk.
func lineCounts(ops []op, side byte) []int {
	counts := make([]int, len(ops)+1)
	for i, o := range ops {
		counts[i+1] = counts[i]
		if o.tag == '=' || o.tag == side {
			counts[i+1]++
		}
	}
	return counts
}

// hunkRange reports the 1-based start line and length, on one side of the
// diff, covered by ops[lo:hi+1]. A zero-length range (a pure insertion or
// deletion) is reported at the line before which it occurs, matching the
// convention used by diff -u.
func hunkRange(counts []int, lo, hi int) (start, length int) {
	length = counts[hi+1] - counts[lo]
	if length == 0 {
		return counts[lo], 0
	}
	return counts[lo] + 1, length
}

func formatRange(start, length int) string {
	if length == 1 {
		return fmt.Sprintf("%d", start)
	}
	return fmt.Sprintf("%d,%d", start, length)
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// unifiedDiff returns a diff -u style comparison of before and after, using
// label as the displayed path on both sides. It returns "" when the two are
// identical. Changed regions are grouped into hunks with up to 3 lines of
// unchanged context, the same as the standard diff tool.
func unifiedDiff(label, before, after string) string {
	if before == after {
		return ""
	}

	a := splitLines(before)
	b := splitLines(after)
	ops := diffLines(a, b)

	var changeIdx []int
	for i, o := range ops {
		if o.tag != '=' {
			changeIdx = append(changeIdx, i)
		}
	}
	if len(changeIdx) == 0 {
		return ""
	}

	const context = 3
	var clusters [][2]int
	start, prev := changeIdx[0], changeIdx[0]
	for _, idx := range changeIdx[1:] {
		if idx-prev-1 > 2*context {
			clusters = append(clusters, [2]int{start, prev})
			start = idx
		}
		prev = idx
	}
	clusters = append(clusters, [2]int{start, prev})

	aCounts := lineCounts(ops, '-')
	bCounts := lineCounts(ops, '+')

	var out strings.Builder
	fmt.Fprintf(&out, "--- a/%s\n+++ b/%s\n", label, label)
	for _, c := range clusters {
		lo, hi := c[0]-context, c[1]+context
		if lo < 0 {
			lo = 0
		}
		if hi >= len(ops) {
			hi = len(ops) - 1
		}

		aStart, aLen := hunkRange(aCounts, lo, hi)
		bStart, bLen := hunkRange(bCounts, lo, hi)
		fmt.Fprintf(&out, "@@ -%s +%s @@\n", formatRange(aStart, aLen), formatRange(bStart, bLen))
		for _, o := range ops[lo : hi+1] {
			out.WriteByte(o.tag)
			out.WriteString(o.text)
			out.WriteByte('\n')
		}
	}
	return out.String()
}
