package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

func processFile(filePath string, moduleName string) (*LocatedDirectives, error) {
	locatedDirectives := NewLocatedDirectives()

	contents, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fileContent := string(contents)
	lines := strings.Split(fileContent, "\n")

	pkgName := extractPackageName(lines)
	if pkgName == "" {
		return nil, fmt.Errorf("%s: no package declaration found", filePath)
	}

	for lineIndex, line := range lines {
		trimmed := strings.TrimSpace(line)
		re := regexp.MustCompile(`^\s*//\s*evercore\s*:\s*([a-zA-Z0-9-]+)\s*$`)
		matches := re.FindStringSubmatch(trimmed)

		if len(matches) > 1 {
			directive := matches[1]
			if err := validateDirective(directive, filePath, lineIndex+1); err != nil {
				return nil, &ErrFileProcessing{
					FilePath: filePath,
					Err:      err,
				}
			}

			nextLine := findNextNonEmptyLine(lines, lineIndex+1)
			structName, err := extractStructName(nextLine, filePath)
			if err != nil {
				return nil, &ErrFileProcessing{
					FilePath: filePath,
					Err:      err,
				}
			}

			pkgPath := filepath.Join(moduleName, filepath.Dir(filePath))
			pkgPath = filepath.ToSlash(pkgPath)

			directiveObj := Directive{
				Type:        DirectiveType(directive),
				Struct:      structName,
				FilePath:    filePath,
				Package:     pkgPath,
				PackageName: pkgName,
				Line:        lineIndex + 1,
			}

			switch directive {
			case string(DirectiveAggregate):
				locatedDirectives.Aggregates[structName] = directiveObj
			case string(DirectiveStateEvent):
				locatedDirectives.StateEvents[structName] = directiveObj
			case string(DirectiveEvent):
				locatedDirectives.Events[structName] = directiveObj
			}
		}
	}

	return &locatedDirectives, nil
}

func extractPackageName(lines []string) string {
	for _, line := range lines {
		if strings.HasPrefix(line, "package ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

func findNextNonEmptyLine(lines []string, start int) string {
	for j := start; j < len(lines); j++ {
		line := strings.TrimSpace(lines[j])
		if line != "" {
			return line
		}
	}
	return ""
}

func extractStructName(line string, filePath string) (string, error) {
	typeRe := regexp.MustCompile(`^type\s+([a-zA-Z0-9_]+)\s+struct\b`)
	matches := typeRe.FindStringSubmatch(line)
	if len(matches) < 2 {
		return "", &ErrInvalidStructDefinition{
			FilePath: filePath,
			Content:  line,
		}
	}

	if !isValidGoIdentifier(matches[1]) {
		return "", &ErrInvalidStructDefinition{
			FilePath: filePath,
			Content:  line,
		}
	}
	return matches[1], nil
}

func isValidGoIdentifier(name string) bool {
	if len(name) == 0 {
		return false
	}

	for i, c := range name {
		if !unicode.IsLetter(c) && c != '_' && (i == 0 || !unicode.IsDigit(c)) {
			return false
		}
	}
	return true
}
