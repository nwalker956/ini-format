# ini-format

A small command-line tool that normalises messy INI files into one
consistent style, the way `gofmt` does for Go source.

## The problem

INI files pile up inconsistencies fast once more than one person or tool
touches them: some lines use `key=value`, others `key = value`, comments
show up as both `;` and `#`, blank lines multiply between edits, and
section headers end up with stray indentation. None of that is wrong per
the (very loose) INI "spec", but it makes diffs noisy and files annoying
to read.

`ini-format` parses that mess and rewrites it with:

- a single `key = value` spacing style
- comments normalised to `; text`, including inline trailing comments
  (`port = 8080 # default` becomes `port = 8080 ; default`)
- quotes around values stripped when they aren't doing anything
  (`name = "simple"` becomes `name = simple`); quotes kept when they
  protect leading/trailing whitespace, an empty value, or a `;`/`#`
  character, and single quotes are switched to double quotes when that's
  safe to do
- at most one blank line between entries
- exactly one blank line before every section header
- no leading or trailing blank lines
- sections with the same name merged into one, in the order they first
  appeared, with each repeat's keys appended after the first occurrence's
- optionally, keys sorted alphabetically within each section (`-sort`);
  a comment written directly above a key (no blank line between them)
  moves with that key

It does not try to validate the file or reject anything: comments,
section headers, and lines it can't parse as `key=value` are all kept
verbatim (just re-indented), so nothing is silently dropped. That includes
a key written twice in the same section (or in two occurrences of the same
`[header]`, since those get merged) - both copies are kept, in order. Run
with `-dupe-check` if you'd rather be told about that than have it happen
quietly.

## Usage

```
go run . config.ini            # print the formatted version to stdout
go run . -w config.ini         # rewrite the file in place
go run . -sort -w config.ini   # also sort keys within each section
go run . -diff config.ini      # show what would change, as a unified diff
go run . -check config.ini     # exit 1 if the file isn't already formatted
go run . -dupe-check config.ini  # exit 1 and list keys that repeat in a section
cat config.ini | go run .      # read from stdin, write to stdout
```

Multiple files can be passed at once; each is formatted independently.
`-check` takes precedence over `-diff` and `-w`: it never writes or prints a
diff, it only reports (one path per line, to stdout) which inputs would
change and exits with status 1 if any would. That makes it suitable for a CI
check that fails the build on unformatted files.

`-dupe-check` takes precedence over all of the above: it never writes,
diffs, or reports formatting status. It only looks for keys that occur more
than once within the same section (matched case-insensitively, the same way
`-sort` compares them) and, for each file, prints one line per repeated key
in the form `path: [section] key appears N times` before exiting with
status 1. A clean file produces no output. If the repeats don't all carry
the same value, the line says so (`... appears N times with differing
values`) - that's the case most worth a second look, since it means the
last one read is silently winning over the rest.

### Config file

If a `.inifmtrc` file exists in the current directory, it sets default
values for `-w`, `-sort`, `-diff`, `-check`, and `-dupe-check`, so a project
can pin its preferred settings instead of everyone remembering the right
flags. It's itself an INI file:

```ini
sort = true
w = true
```

Any flag actually given on the command line overrides the config file. Use
`-config path/to/file` to read defaults from somewhere other than
`.inifmtrc` in the current directory.

### Example

Input:

```ini
[server]
   host=127.0.0.1
port =8080

  # listen on both protocols
enable_ipv6=true



[logging]
level = debug
# where to write logs
path=/var/log/app.log
```

Output of `go run . config.ini`:

```ini
[server]
host = 127.0.0.1
port = 8080

; listen on both protocols
enable_ipv6 = true

[logging]
level = debug
; where to write logs
path = /var/log/app.log
```

## Building

Standard library only, no dependencies:

```
go build -o ini-format .
```
