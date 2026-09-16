package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

// errNotFormatted signals, from processFile back to main, that -check found
// input that would be changed by formatting. It isn't a failure to read or
// parse anything, so it's reported differently from a normal error.
var errNotFormatted = errors.New("not formatted")

// errDuplicateKeys signals, from processFile back to main, that -dupe-check
// found one or more keys repeated within the same section.
var errDuplicateKeys = errors.New("duplicate keys")

func main() {
	write := flag.Bool("w", false, "write result back to the file instead of stdout")
	doSort := flag.Bool("sort", false, "sort keys alphabetically within each section")
	showDiff := flag.Bool("diff", false, "print a diff instead of writing or printing the result")
	check := flag.Bool("check", false, "exit with status 1 if input is not already formatted; writes nothing")
	dupeCheck := flag.Bool("dupe-check", false, "report keys that repeat within a section and exit with status 1 if any are found; writes nothing")
	configPath := flag.String("config", "", "path to a config file setting default flags (default: .inifmtrc in the current directory, if present)")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "inifmt:", err)
		os.Exit(1)
	}
	applyConfig(&cfg, write, doSort, showDiff, check, dupeCheck)

	args := flag.Args()
	if len(args) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "inifmt:", err)
			os.Exit(1)
		}
		if *dupeCheck {
			dupes, err := FindDuplicateKeys(bytes.NewReader(data))
			if err != nil {
				fmt.Fprintln(os.Stderr, "inifmt:", err)
				os.Exit(1)
			}
			for _, d := range dupes {
				fmt.Printf("<standard input>: %s\n", d)
			}
			if len(dupes) > 0 {
				os.Exit(1)
			}
			return
		}

		out, err := Format(bytes.NewReader(data), *doSort)
		if err != nil {
			fmt.Fprintln(os.Stderr, "inifmt:", err)
			os.Exit(1)
		}
		switch {
		case *check:
			if out != string(data) {
				fmt.Println("<standard input>")
				os.Exit(1)
			}
		case *showDiff:
			fmt.Print(unifiedDiff("<standard input>", string(data), out))
		default:
			fmt.Print(out)
		}
		return
	}

	status := 0
	for _, path := range args {
		switch err := processFile(path, *write, *doSort, *showDiff, *check, *dupeCheck); {
		case errors.Is(err, errNotFormatted):
			fmt.Println(path)
			status = 1
		case errors.Is(err, errDuplicateKeys):
			status = 1
		case err != nil:
			fmt.Fprintf(os.Stderr, "inifmt: %s: %v\n", path, err)
			status = 1
		}
	}
	os.Exit(status)
}

// applyConfig fills in any flag that wasn't given explicitly on the command
// line with the corresponding value from cfg. Flags the user did set take
// priority over the config file, which is why this runs after flag.Parse
// rather than being used to seed the flag defaults.
func applyConfig(cfg *config, write, doSort, showDiff, check, dupeCheck *bool) {
	set := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })

	if !set["w"] {
		*write = cfg.write
	}
	if !set["sort"] {
		*doSort = cfg.sort
	}
	if !set["diff"] {
		*showDiff = cfg.diff
	}
	if !set["check"] {
		*check = cfg.check
	}
	if !set["dupe-check"] {
		*dupeCheck = cfg.dupeCheck
	}
}

// processFile reads path and, depending on which mode is requested, either
// reports duplicate keys, checks formatting, prints a diff, writes the
// formatted result back, or prints it to stdout. dupeCheck takes precedence
// over every other mode: like check, it never writes or prints a diff, it
// only reports.
func processFile(path string, write, doSort, showDiff, check, dupeCheck bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if dupeCheck {
		dupes, err := FindDuplicateKeys(bytes.NewReader(data))
		if err != nil {
			return err
		}
		for _, d := range dupes {
			fmt.Printf("%s: %s\n", path, d)
		}
		if len(dupes) > 0 {
			return errDuplicateKeys
		}
		return nil
	}

	out, err := Format(bytes.NewReader(data), doSort)
	if err != nil {
		return err
	}

	switch {
	case check:
		if string(data) != out {
			return errNotFormatted
		}
		return nil
	case showDiff:
		_, err = fmt.Print(unifiedDiff(path, string(data), out))
		return err
	case write:
		if string(data) == out {
			return nil
		}
		return os.WriteFile(path, []byte(out), 0644)
	default:
		_, err = fmt.Print(out)
		return err
	}
}
