package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func walkProject(moduleName string) (LocatedDirectives, error) {
	locatedDirectives := NewLocatedDirectives()

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".go" {
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
		}
		return nil
	})

	return locatedDirectives, err
}
