package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	write := flag.Bool("w", false, "write result back to the file instead of stdout")
	doSort := flag.Bool("sort", false, "sort keys alphabetically within each section")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		out, err := Format(os.Stdin, *doSort)
		if err != nil {
			fmt.Fprintln(os.Stderr, "inifmt:", err)
			os.Exit(1)
		}
		fmt.Print(out)
		return
	}

	status := 0
	for _, path := range args {
		if err := processFile(path, *write, *doSort); err != nil {
			fmt.Fprintf(os.Stderr, "inifmt: %s: %v\n", path, err)
			status = 1
		}
	}
	os.Exit(status)
}

func processFile(path string, write, doSort bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	out, ferr := Format(f, doSort)
	f.Close()
	if ferr != nil {
		return ferr
	}
	if write {
		return os.WriteFile(path, []byte(out), 0644)
	}
	_, err = fmt.Print(out)
	return err
}
