package main

import "fmt"

type ErrInvalidDirective struct {
    Directive string
    FilePath  string
    Line      int
}

func (e ErrInvalidDirective) Error() string {
    return fmt.Sprintf("%s:%d: invalid evercore directive '%s'", e.FilePath, e.Line, e.Directive)
}

type ErrInvalidStructDefinition struct {
    FilePath string
    Line     int
    Content  string
}

func (e ErrInvalidStructDefinition) Error() string {
    return fmt.Sprintf("%s:%d: evercore directive must be followed by a struct type definition, got: %q",
        e.FilePath, e.Line, e.Content)
}

type ErrFileProcessing struct {
    FilePath string
    Err      error
}

func (e ErrFileProcessing) Error() string {
    return fmt.Sprintf("processing %s: %v", e.FilePath, e.Err)
}

func (e ErrFileProcessing) Unwrap() error {
    return e.Err
}
