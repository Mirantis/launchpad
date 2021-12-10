package launchpad

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var stderr = os.Stderr

// New launchpad terraform provider
func New() tfsdk.Provider {
	return &provider{}
}

type provider struct {
	configured bool
}

// GetSchema
func (p *provider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"placeholder": {
				Type:     types.StringType,
				Description: "Unused attribute placeholder, as an empty set is not allowed.",
				Optional: true,
				Computed: true,
			},
		},
	}, nil
}

// Provider schema struct
type providerData struct {
	Username types.String `tfsdk:"username"`
	Host     types.String `tfsdk:"host"`
	Password types.String `tfsdk:"password"`
}

func (p *provider) Configure(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
	// Retrieve provider data from configuration
	p.configured = true
}

// GetResources - Defines provider resources
func (p *provider) GetResources(_ context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"launchpad_product_fromyaml": resourceProductYamlType{},
	}, nil
}

// GetDataSources - Defines provider data sources
func (p *provider) GetDataSources(_ context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{}, nil
}
