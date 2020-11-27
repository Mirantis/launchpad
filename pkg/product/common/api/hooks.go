package api

// BeforeAfter is the a child struct for the Hooks struct, containing sections for Before and After
type BeforeAfter struct {
	Before *[]string `yaml:"before" default:"[]"`
	After  *[]string `yaml:"after" default:"[]"`
}

// Hooks is a list of hook-points
type Hooks struct {
	Apply *BeforeAfter `yaml:"apply" default:"{}"`
	Reset *BeforeAfter `yaml:"reset" default:"{}"`
}
