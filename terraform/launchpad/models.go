package launchpad

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ProductFromYaml a product definition using raw yaml
type ProductFromYaml struct {
	Yaml        types.String `tfsdk:"yaml"`
}
