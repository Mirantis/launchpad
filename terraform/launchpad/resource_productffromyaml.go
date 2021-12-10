package launchpad

import (
	"context"
	"time"

	event "gopkg.in/segmentio/analytics-go.v3"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
)

type resourceProductYamlType struct{}

// Product Resource schema
func (r resourceProductYamlType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"yaml": {
				MarkdownDescription: "Yaml string contents to send to launchpad",
				Required:            true,
				Type:                types.StringType,
			},
		},
	}, nil
}

// New resource instance
func (r resourceProductYamlType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceProductYaml{
		p: *(p.(*provider)),
	}, nil
}

type resourceProductYaml struct {
	p provider
}

// Create a new resource
func (r resourceProductYaml) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var stateProduct ProductFromYaml

	diags := req.Plan.Get(ctx, &stateProduct)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	launchpadProductApplyFromYaml(&stateProduct, resp.Diagnostics)

	// Set state
	diags = resp.State.Set(ctx, &stateProduct)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r resourceProductYaml) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	/**
		The Product object has a Describe method which could be used to determine
		state of the cluster, but the describe method is particular to the actual
		project class, ain that it takes a string key for the report type.
		This means that we have to effectively know what reports are available
		through analysis.
		Additionally, the describe processes really just rely on fmt output, so
		there is no space for retrieval using the existing code.

		This is a weakness in the mcc code.
	*/
}

// Update resource if there is any change in the yaml.
func (r resourceProductYaml) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var planProduct ProductFromYaml
	var stateProduct ProductFromYaml

	diags := req.Plan.Get(ctx, &planProduct)
	resp.Diagnostics.Append(diags...)
	diags = req.Plan.Get(ctx, &stateProduct)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

  // only run the apply if the yaml has changed.
	if stateProduct.Yaml.Value != planProduct.Yaml.Value {
		launchpadProductApplyFromYaml(&stateProduct, resp.Diagnostics)
	}

	// Set state
	diags = resp.State.Set(ctx, &planProduct)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func (r resourceProductYaml) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var planProduct ProductFromYaml

	diags := req.State.Get(ctx, &planProduct)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// unpassed variable will trigger a diagnostic error, so no need to test it.
	yaml := planProduct.Yaml.Value
	product, err := config.ProductFromYAML([]byte(yaml))

	start := time.Now()
	analytics.TrackEvent("Cluster Reset Started", nil)

	err = product.Reset()

	if err != nil {
		analytics.TrackEvent("Cluster Apply Failed", nil)
		resp.Diagnostics.AddError(
			"Error resetting launchpad from yaml",
			err.Error(),
		)
	} else {
		duration := time.Since(start)
		props := event.Properties{
			"duration": duration.Seconds(),
		}
		analytics.TrackEvent("Cluster Reset Completed", props)
	}

	planProduct.Yaml = types.String{Value:"", Null:true, Unknown:true}

	// Set state
	diags = resp.State.Set(ctx, &planProduct)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// run launchpad apply using a product from yaml
func launchpadProductApplyFromYaml(prodFromYaml *ProductFromYaml, diagnostics diag.Diagnostics) error {
	// unpassed variable will trigger a diagnostic error, so no need to test it.
	yaml := prodFromYaml.Yaml.Value
	product, err := config.ProductFromYAML([]byte(yaml))

	start := time.Now()
	analytics.TrackEvent("Cluster Apply Started", nil)

	err = product.Apply(false, false)

	if err != nil {
		analytics.TrackEvent("Cluster Apply Failed", nil)
		diagnostics.AddError(
			"Error running launchpad from yaml",
			err.Error(),
		)
	} else {
		duration := time.Since(start)
		props := event.Properties{
			"duration": duration.Seconds(),
		}
		analytics.TrackEvent("Cluster Apply Completed", props)
	}

	// prodFromYaml.ClusterName = types.String{Value:product.ClusterName()}
	// prodFromYaml.LastUpdated = types.String{Value:start.Format("RFC3339")}

	return err
}