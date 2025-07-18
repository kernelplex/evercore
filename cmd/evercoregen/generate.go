package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"text/template"
)

func GenerateCode(outputData OutputData) error {
	if outputData.Config.Verbose {
		log.Printf("Starting code generation")
		log.Printf("Found %d aggregates, %d events, %d state events",
			len(outputData.Aggregates),
			len(outputData.Events),
			len(outputData.StateEvents))
	}
	uniquePackages := make(map[string]struct{})

	// Helper function to collect values from a map
	collectValues := func(m map[string]Directive) {
		for _, directive := range m {
			if directive.Type == "aggregate" {
				continue
			}
			uniquePackages[directive.Package] = struct{}{}
		}
	}

	// Collect from all three maps
	collectValues(outputData.Aggregates)
	collectValues(outputData.Events)
	collectValues(outputData.StateEvents)

	// Convert to slice
	var uniqueImports []string
	for packagePath := range uniquePackages {
		uniqueImports = append(uniqueImports, packagePath)
	}

	// Sort for deterministic output
	sort.Strings(uniqueImports)

	// Store in outputData
	outputData.UniqueImports = uniqueImports

	// Generate the output file using template
	main_tmpl, err := template.ParseFS(templateFiles, "templates/targetcode.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	eventTmpl, err := template.ParseFS(templateFiles, "templates/events.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	aggregateTmpl, err := template.ParseFS(templateFiles, "templates/aggregates.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	outputFile := filepath.Join(outputData.Config.OutputDir, "generated.go")
	if outputData.Config.Verbose {
		log.Printf("Creating main output file: %s", outputFile)
	}
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	err = main_tmpl.Execute(f, outputData)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Create subdirectories for events and aggregates
	eventsDir := filepath.Join(outputData.Config.OutputDir, "events")
	if outputData.Config.Verbose {
		log.Printf("Creating events directory: %s", eventsDir)
	}
	if err := os.MkdirAll(eventsDir, 0755); err != nil {
		return fmt.Errorf("failed to create events directory: %w", err)
	}

	aggregatesDir := filepath.Join(outputData.Config.OutputDir, "aggregates")
	if outputData.Config.Verbose {
		log.Printf("Creating aggregates directory: %s", aggregatesDir)
	}
	if err := os.MkdirAll(aggregatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create aggregates directory: %w", err)
	}

	// Generate aggregates.go file
	aggregatesFile := filepath.Join(aggregatesDir, "aggregates.go")
	af, err := os.Create(aggregatesFile)
	if err != nil {
		return fmt.Errorf("failed to create aggregates file: %w", err)
	}
	defer af.Close()

	err = aggregateTmpl.Execute(af, outputData)
	if err != nil {
		return fmt.Errorf("failed to execute aggregates template: %w", err)
	}

	// Generate events.go file
	eventsFile := filepath.Join(eventsDir, "events.go")
	ef, err := os.Create(eventsFile)
	if err != nil {
		return fmt.Errorf("failed to create events file: %w", err)
	}
	defer ef.Close()

	err = eventTmpl.Execute(ef, outputData)
	if err != nil {
		return fmt.Errorf("failed to execute events template: %w", err)
	}

	return nil
}
