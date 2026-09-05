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

func main() {
	write := flag.Bool("w", false, "write result back to the file instead of stdout")
	doSort := flag.Bool("sort", false, "sort keys alphabetically within each section")
	showDiff := flag.Bool("diff", false, "print a diff instead of writing or printing the result")
	check := flag.Bool("check", false, "exit with status 1 if input is not already formatted; writes nothing")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "inifmt:", err)
			os.Exit(1)
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
		switch err := processFile(path, *write, *doSort, *showDiff, *check); {
		case errors.Is(err, errNotFormatted):
			fmt.Println(path)
			status = 1
		case err != nil:
			fmt.Fprintf(os.Stderr, "inifmt: %s: %v\n", path, err)
			status = 1
		}
	}
	os.Exit(status)
}

func processFile(path string, write, doSort, showDiff, check bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
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
