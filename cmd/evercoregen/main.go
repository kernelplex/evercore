package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

//go:embed templates/*
var templateFiles embed.FS

type OutputConfig struct {
	OutputDir string `json:"output_dir"`
	OutputPkg string `json:"output_pkg"`
	Verbose   bool   `json:"verbose"`
}

type OutputData struct {
	LocatedDirectives
	Config        OutputConfig `json:"config"`
	UniqueImports []string     `json:"unique_imports,omitempty"`
	ModuleName    string       `json:"module_name"`
}

func main() {
	var (
		outputDir   string
		outputPkg   string
		verbose     bool
		showVersion bool
	)

	flag.StringVar(&outputDir, "output-dir", "", "Directory to write generated files (required)")
	flag.StringVar(&outputPkg, "output-pkg", "", "Go package name for generated files (required)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&showVersion, "version", false, "Show version information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "\nOptions:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExample:")
		fmt.Fprintf(os.Stderr, "  %s -output-dir=internal/generated -output-pkg=generated\n", os.Args[0])
	}

	flag.Parse()

	// Validate required flags
	if outputDir == "" || outputPkg == "" {
		flag.Usage()
		os.Exit(2)
	}

	// Show help if requested
	if len(flag.Args()) > 0 {
		flag.Usage()
		os.Exit(2)
	}

	// Create output directory if it doesn't exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatalf("failed to create output directory: %v", err)
		}
	}

	// Clean the output directory path
	outputDir, err := filepath.Abs(outputDir)
	if err != nil {
		log.Fatalf("invalid output directory path: %v", err)
	}
	moduleName, err := getModuleName()
	if err != nil {
		log.Fatal(err)
	}

	config := DefaultConfig()
	config.OutputDir = outputDir
	config.OutputPkg = outputPkg
	config.Verbose = verbose

	if verbose {
		log.Printf("Verbose mode enabled")
		log.Printf("Output directory: %s", outputDir)
		log.Printf("Output package: %s", outputPkg)
	}

	locatedDirectives, err := walkProject(moduleName, config)
	if err != nil {
		var fileErr *ErrFileProcessing
		if errors.As(err, &fileErr) {
			log.Printf("Error processing file: %v", fileErr)
		} else {
			log.Fatalf("Fatal error: %v", err)
		}
		os.Exit(1)
	}

	output := OutputData{
		Config: OutputConfig{
			OutputDir: outputDir,
			OutputPkg: outputPkg,
			Verbose:   verbose,
		},
		ModuleName: moduleName,
	}
	output.LocatedDirectives = locatedDirectives
	output.Aggregates = locatedDirectives.Aggregates
	output.StateEvents = locatedDirectives.StateEvents
	output.Events = locatedDirectives.Events

	err = GenerateCode(output)
	if err != nil {
		log.Fatal(err)
	}
}
