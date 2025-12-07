package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	root := flag.String("root", "", "root directory to start")
	list := flag.Bool("list", false, "list files")
	ext := flag.String("ext", "", "file extensions to filter")
	size := flag.Int64("min", 0, "minimum file size")
	del := flag.String("del", "", "files to delete")
	logFile := flag.String("log", "", "log file path (default: stdout)")
	flag.Parse()

	cfg, err := ParseConfig(*list, *ext, *del, *size, *logFile)
	if err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}

	if err := ValidateConfig(cfg); err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}

	if err := ValidateRoot(*root); err != nil {
		fmt.Printf("Root directory error: %v\n", err)
		os.Exit(1)
	}

	if err := run(*root, os.Stdout, cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func run(root string, out *os.File, cfg Config) error {
	processor := NewFileProcessor(root, cfg)
	return processor.ProcessFiles(out)
}
