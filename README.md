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
- comments normalised to `; text`
- at most one blank line between entries
- exactly one blank line before every section header
- no leading or trailing blank lines
- optionally, keys sorted alphabetically within each section (`-sort`)

It does not try to validate the file or reject anything: comments,
section headers, and lines it can't parse as `key=value` are all kept
verbatim (just re-indented), so nothing is silently dropped.

## Usage

```
go run . config.ini            # print the formatted version to stdout
go run . -w config.ini         # rewrite the file in place
go run . -sort -w config.ini   # also sort keys within each section
cat config.ini | go run .      # read from stdin, write to stdout
```

Multiple files can be passed at once; each is formatted independently.

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

## Current limitations

- Inline (trailing) comments on a `key = value` line are treated as part
  of the value, not stripped out.
- Quoting style of values is left untouched.
- With `-sort`, comments stay in their original position in the file
  rather than moving with the key they were written above.

## Building

Standard library only, no dependencies:

```
go build -o ini-format .
```
