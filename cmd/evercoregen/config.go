package main

type Config struct {
	OutputDir    string   `yaml:"output_dir" json:"output_dir"`
	OutputPkg    string   `yaml:"output_pkg" json:"output_pkg"`
	ExcludeDirs  []string `yaml:"exclude_dirs" json:"exclude_dirs"`
	IncludeGlobs []string `yaml:"include_globs" json:"include_globs"`
	Verbose      bool     `yaml:"verbose" json:"verbose"`
}

func DefaultConfig() Config {
	return Config{
		ExcludeDirs:  []string{"vendor", "testdata", ".*"},
		IncludeGlobs: []string{"**/*.go"},
	}
}
