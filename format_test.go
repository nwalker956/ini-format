package main

import (
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	cases := []struct {
		name string
		in   string
		sort bool
		want string
	}{
		{
			name: "empty input",
			in:   "",
			want: "",
		},
		{
			name: "only blank lines",
			in:   "\n\n   \n\n",
			want: "",
		},
		{
			name: "normalises key spacing",
			in:   "host=127.0.0.1\nport =8080\n  timeout= 30\n",
			want: "host = 127.0.0.1\nport = 8080\ntimeout = 30\n",
		},
		{
			name: "normalises comment markers",
			in:   "; already ok\n# hash style\n;\n",
			want: "; already ok\n; hash style\n;\n",
		},
		{
			name: "collapses blank runs and trims edges",
			in:   "\n\nkey1=a\n\n\n\nkey2=b\n\n\n",
			want: "key1 = a\n\nkey2 = b\n",
		},
		{
			name: "inserts one blank line before each section header",
			in:   "[a]\nx=1\n[b]\ny=2\n",
			want: "[a]\nx = 1\n\n[b]\ny = 2\n",
		},
		{
			name: "global section content precedes first header with a blank line",
			in:   "greeting=hi\n[server]\nhost=1\n",
			want: "greeting = hi\n\n[server]\nhost = 1\n",
		},
		{
			name: "keeps unparsable lines verbatim",
			in:   "weird line here\nkey=val\n",
			want: "weird line here\nkey = val\n",
		},
		{
			name: "splits value on the first equals only",
			in:   `path=C:\foo=bar` + "\n",
			want: "path = C:\\foo=bar\n",
		},
		{
			name: "section header whitespace is trimmed",
			in:   "[  server  ]\nhost=1\n",
			want: "[server]\nhost = 1\n",
		},
		{
			name: "sort orders keys case-insensitively within a section",
			in:   "[a]\nZebra=1\napple=2\nMango=3\n",
			sort: true,
			want: "[a]\napple = 2\nMango = 3\nZebra = 1\n",
		},
		{
			name: "sort keeps a comment attached to the key directly below it",
			in:   "[a]\n; about zebra\nzebra=1\napple=2\n",
			sort: true,
			want: "[a]\napple = 2\n; about zebra\nzebra = 1\n",
		},
		{
			name: "sort keeps a run of comments attached to the key below them",
			in:   "[a]\n; first note\n; second note\nzebra=1\napple=2\n",
			sort: true,
			want: "[a]\napple = 2\n; first note\n; second note\nzebra = 1\n",
		},
		{
			name: "sort leaves a comment in place when a blank line separates it from the key below",
			in:   "[a]\n; general note\n\nzebra=1\napple=2\n",
			sort: true,
			want: "[a]\n; general note\n\napple = 2\nzebra = 1\n",
		},
		{
			name: "sort leaves a trailing comment with no key below it in place",
			in:   "[a]\nzebra=1\napple=2\n; trailing note\n",
			sort: true,
			want: "[a]\napple = 2\nzebra = 1\n; trailing note\n",
		},
		{
			name: "extracts inline trailing comment",
			in:   "port=8080 ; default port\n",
			want: "port = 8080 ; default port\n",
		},
		{
			name: "extracts inline trailing comment with hash marker",
			in:   "port=8080 # default port\n",
			want: "port = 8080 ; default port\n",
		},
		{
			name: "inline comment with no space before marker",
			in:   "port=8080;default port\n",
			want: "port = 8080 ; default port\n",
		},
		{
			name: "quoted value keeps marker characters literal",
			in:   `path="C:\later;dir"` + "\n",
			want: `path = "C:\later;dir"` + "\n",
		},
		{
			name: "quoted value can still have a real inline comment after it",
			in:   `name="a;b" ; that's the name` + "\n",
			want: `name = "a;b" ; that's the name` + "\n",
		},
		{
			name: "unnecessary double quotes are stripped",
			in:   `name="simple"` + "\n",
			want: "name = simple\n",
		},
		{
			name: "unnecessary single quotes are stripped",
			in:   "name='simple'\n",
			want: "name = simple\n",
		},
		{
			name: "single quotes protecting whitespace are converted to double",
			in:   "name=' padded '\n",
			want: `name = " padded "` + "\n",
		},
		{
			name: "single quotes are kept when converting would change meaning",
			in:   `name=' "quoted" padded '` + "\n",
			want: `name = ' "quoted" padded '` + "\n",
		},
		{
			name: "empty quoted value is kept quoted",
			in:   `name=""` + "\n",
			want: `name = ""` + "\n",
		},
		{
			name: "repeated section header merges into one section",
			in:   "[a]\nx=1\n[b]\ny=2\n[a]\nz=3\n",
			want: "[a]\nx = 1\nz = 3\n\n[b]\ny = 2\n",
		},
		{
			name: "merged section keeps sort scoped to the combined items",
			in:   "[a]\nzebra=1\n[a]\napple=2\n",
			sort: true,
			want: "[a]\napple = 2\nzebra = 1\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Format(strings.NewReader(tc.in), tc.sort)
			if err != nil {
				t.Fatalf("Format returned error: %v", err)
			}
			if got != tc.want {
				t.Errorf("Format(%q, sort=%v):\n got: %q\nwant: %q", tc.in, tc.sort, got, tc.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	sections, err := parse(strings.NewReader("intro=1\n[db]\n; note\nhost = localhost\n\nport=5432\nnot a kv line\n"))
	if err != nil {
		t.Fatalf("parse returned error: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("got %d sections, want 2", len(sections))
	}

	global := sections[0]
	if global.name != "" {
		t.Errorf("global section name = %q, want empty", global.name)
	}
	if len(global.items) != 1 || global.items[0].kind != kindEntry || global.items[0].key != "intro" || global.items[0].value != "1" {
		t.Errorf("global items = %+v, want single intro=1 entry", global.items)
	}

	db := sections[1]
	if db.name != "db" {
		t.Errorf("second section name = %q, want %q", db.name, "db")
	}
	wantKinds := []lineKind{kindComment, kindEntry, kindBlank, kindEntry, kindRaw}
	if len(db.items) != len(wantKinds) {
		t.Fatalf("db has %d items, want %d: %+v", len(db.items), len(wantKinds), db.items)
	}
	for i, k := range wantKinds {
		if db.items[i].kind != k {
			t.Errorf("db.items[%d].kind = %v, want %v", i, db.items[i].kind, k)
		}
	}
	if db.items[1].key != "host" || db.items[1].value != "localhost" {
		t.Errorf("host entry = %+v", db.items[1])
	}
	if db.items[4].text != "not a kv line" {
		t.Errorf("raw line text = %q, want %q", db.items[4].text, "not a kv line")
	}
}

func TestCollapseBlanks(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "nil input", in: nil, want: []string{}},
		{name: "all blank", in: []string{"", "", ""}, want: []string{}},
		{
			name: "leading and trailing blanks stripped",
			in:   []string{"", "", "a", "b", "", ""},
			want: []string{"a", "b"},
		},
		{
			name: "internal runs squashed to one",
			in:   []string{"a", "", "", "", "b"},
			want: []string{"a", "", "b"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := collapseBlanks(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("collapseBlanks(%v) = %v, want %v", tc.in, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("collapseBlanks(%v)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
				}
			}
		})
	}
}
