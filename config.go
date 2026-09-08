package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// config holds default flag values loaded from an inifmt config file. Any
// flag given explicitly on the command line overrides the corresponding
// config value.
type config struct {
	sort  bool
	write bool
	diff  bool
	check bool
}

// defaultConfigName is the file inifmt looks for in the current directory
// when no -config flag is given.
const defaultConfigName = ".inifmtrc"

// loadConfig reads flag defaults from an inifmt config file. The file is
// itself INI syntax, with top-level keys named after their flags (w, sort,
// diff, check) set to true or false.
//
// A missing file at the default location is not an error, since most
// directories won't have one. A missing file named explicitly with path is
// an error, since the user asked for it by name.
func loadConfig(path string) (config, error) {
	var cfg config
	explicit := path != ""
	if !explicit {
		path = defaultConfigName
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return cfg, nil
		}
		return cfg, err
	}

	sections, err := parse(strings.NewReader(string(data)))
	if err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}

	for _, s := range sections {
		for _, it := range s.items {
			if it.kind != kindEntry {
				continue
			}
			b, err := strconv.ParseBool(it.value)
			if err != nil {
				return cfg, fmt.Errorf("%s: key %q: %w", path, it.key, err)
			}
			switch strings.ToLower(it.key) {
			case "sort":
				cfg.sort = b
			case "w":
				cfg.write = b
			case "diff":
				cfg.diff = b
			case "check":
				cfg.check = b
			default:
				return cfg, fmt.Errorf("%s: unknown key %q", path, it.key)
			}
		}
	}
	return cfg, nil
}
