package main

import (
	"testing"
	"text/template"
)

func TestTemplatesParse(t *testing.T) {
	tests := []struct {
		name     string
		template string
	}{
		{"main template", "templates/targetcode.tmpl"},
		{"events template", "templates/events.tmpl"},
		{"aggregates template", "templates/aggregates.tmpl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := template.ParseFS(templateFiles, tt.template)
			if err != nil {
				t.Errorf("Failed to parse template %s: %v", tt.template, err)
			}
		})
	}
}
