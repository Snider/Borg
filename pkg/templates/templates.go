package templates

// Step represents a single step in a collection template.
type Step struct {
	Collect string `yaml:"collect"`
	Output  string `yaml:"output"`
	Encrypt bool   `yaml:"encrypt"`
	Depth   int    `yaml:"depth"`
}

// Template represents a collection template.
type Template struct {
	Name      string            `yaml:"name"`
	Steps     []Step            `yaml:"steps"`
	Variables map[string]string `yaml:"variables"`
}
