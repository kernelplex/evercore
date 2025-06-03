package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

func getModuleName() (string, error) {
	modBytes, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	modFile := string(modBytes)
	lines := strings.Split(modFile, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				return "", fmt.Errorf("invalid module declaration in go.mod")
			}
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("no module declaration found in go.mod")
}

func shouldProcessFile(path string, config Config) bool {
	// Convert path to forward slashes for consistent matching
	normalizedPath := filepath.ToSlash(path)

	// Check against exclude patterns first
	for _, pattern := range config.ExcludeDirs {
		matched, err := doublestar.Match(pattern, normalizedPath)
		if err != nil {
			log.Printf("Invalid exclude pattern %q: %v", pattern, err)
			continue
		}
		if matched {
			if config.Verbose {
				log.Printf("Excluding file %s (matches exclude pattern %s)", path, pattern)
			}
			return false
		}
	}

	// Check include patterns
	for _, pattern := range config.IncludeGlobs {
		matched, err := doublestar.Match(pattern, normalizedPath)
		if err != nil {
			log.Printf("Invalid include pattern %q: %v", pattern, err)
			continue
		}
		if matched {
			if config.Verbose {
				log.Printf("Including file %s (matches include pattern %s)", path, pattern)
			}
			return true
		}
	}

	if config.Verbose {
		log.Printf("Excluding file %s (no include patterns matched)", path)
	}
	return false
}

func walkProject(moduleName string, config Config) (LocatedDirectives, error) {
	if config.Verbose {
		log.Printf("Starting project scan")
		log.Printf("Module: %s", moduleName)
		log.Printf("Include patterns: %v", config.IncludeGlobs)
		log.Printf("Exclude patterns: %v", config.ExcludeDirs)
	}
	startTime := time.Now()
	var filesProcessed, filesSkipped int
	locatedDirectives := NewLocatedDirectives()

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if shouldProcessFile(path, config) {
			filesProcessed++
			newDirectives, err := processFile(path, moduleName)
			if err != nil {
				return err
			}

			// Merge new directives
			for k, v := range newDirectives.Aggregates {
				locatedDirectives.Aggregates[k] = v
			}
			for k, v := range newDirectives.StateEvents {
				locatedDirectives.StateEvents[k] = v
			}
			for k, v := range newDirectives.Events {
				locatedDirectives.Events[k] = v
			}
		} else {
			filesSkipped++
		}
		return nil
	})

	// Always log basic stats
	stats := struct {
		Duration       time.Duration `json:"duration"`
		FilesProcessed int           `json:"files_processed"`
		FilesSkipped   int           `json:"files_skipped"`
		Aggregates     int           `json:"aggregates"`
		Events         int           `json:"events"`
		StateEvents    int           `json:"state_events"`
	}{
		Duration:       time.Since(startTime),
		FilesProcessed: filesProcessed,
		FilesSkipped:   filesSkipped,
		Aggregates:     len(locatedDirectives.Aggregates),
		Events:         len(locatedDirectives.Events),
		StateEvents:    len(locatedDirectives.StateEvents),
	}

	jsonData, _ := json.Marshal(stats)
	log.Printf("Processed project in %s - %s", stats.Duration, jsonData)

	if config.Verbose {
		debugData := struct {
			ModuleName string            `json:"module"`
			Config     Config            `json:"config"`
			Stats      interface{}       `json:"stats"`
			Directives LocatedDirectives `json:"directives"`
			Error      error             `json:"error,omitempty"`
		}{
			ModuleName: moduleName,
			Config:     config,
			Stats:      stats,
			Directives: locatedDirectives,
			Error:      err,
		}

		jsonData, _ := json.MarshalIndent(debugData, "", "  ")
		log.Printf("DEBUG - Detailed results:\n%s", jsonData)
	}

	return locatedDirectives, err
}
