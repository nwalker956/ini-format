package main

import "testing"

func TestUnifiedDiff(t *testing.T) {
	cases := []struct {
		name   string
		before string
		after  string
		want   string
	}{
		{
			name:   "identical content",
			before: "a\nb\nc\n",
			after:  "a\nb\nc\n",
			want:   "",
		},
		{
			name:   "single line changed",
			before: "a\nb\nc\n",
			after:  "a\nx\nc\n",
			want:   "--- a/test.ini\n+++ b/test.ini\n@@ -1,3 +1,3 @@\n a\n-b\n+x\n c\n",
		},
		{
			name:   "line inserted",
			before: "a\nb\n",
			after:  "a\nx\nb\n",
			want:   "--- a/test.ini\n+++ b/test.ini\n@@ -1,2 +1,3 @@\n a\n+x\n b\n",
		},
		{
			name:   "line removed",
			before: "a\nx\nb\n",
			after:  "a\nb\n",
			want:   "--- a/test.ini\n+++ b/test.ini\n@@ -1,3 +1,2 @@\n a\n-x\n b\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := unifiedDiff("test.ini", tc.before, tc.after)
			if got != tc.want {
				t.Errorf("unifiedDiff(%q, %q):\n got: %q\nwant: %q", tc.before, tc.after, got, tc.want)
			}
		})
	}
}

func TestUnifiedDiffChangesFarApartGetSeparateHunks(t *testing.T) {
	before := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n14\n15\n"
	after := "x\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n14\ny\n"

	got := unifiedDiff("test.ini", before, after)
	want := "--- a/test.ini\n+++ b/test.ini\n" +
		"@@ -1,4 +1,4 @@\n-1\n+x\n 2\n 3\n 4\n" +
		"@@ -12,4 +12,4 @@\n 12\n 13\n 14\n-15\n+y\n"
	if got != want {
		t.Errorf("unifiedDiff produced:\n%s\nwant:\n%s", got, want)
	}
}
