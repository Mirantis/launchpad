package main

import (
	"context"
	"github.com/Mirantis/mcc/terraform/launchpad"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

func main() {
	tfsdk.Serve(context.Background(), launchpad.New, tfsdk.ServeOpts{
		Name: "launchpad",
	})
}
