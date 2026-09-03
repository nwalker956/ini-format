package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	write := flag.Bool("w", false, "write result back to the file instead of stdout")
	doSort := flag.Bool("sort", false, "sort keys alphabetically within each section")
	showDiff := flag.Bool("diff", false, "print a diff instead of writing or printing the result")
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
		if *showDiff {
			fmt.Print(unifiedDiff("<standard input>", string(data), out))
		} else {
			fmt.Print(out)
		}
		return
	}

	status := 0
	for _, path := range args {
		if err := processFile(path, *write, *doSort, *showDiff); err != nil {
			fmt.Fprintf(os.Stderr, "inifmt: %s: %v\n", path, err)
			status = 1
		}
	}
	os.Exit(status)
}

func processFile(path string, write, doSort, showDiff bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := Format(bytes.NewReader(data), doSort)
	if err != nil {
		return err
	}

	switch {
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
