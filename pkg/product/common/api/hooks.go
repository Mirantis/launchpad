package api

// BeforeAfter is the a child struct for the Hooks struct, containing sections for Before and After
type BeforeAfter struct {
	Before *[]string `yaml:"before,omitempty"`
	After  *[]string `yaml:"after,omitempty"`
}

// Hooks is a list of hook-points
type Hooks struct {
	Apply *BeforeAfter `yaml:"apply,omitempty"`
	Reset *BeforeAfter `yaml:"reset,omitempty"`
}
