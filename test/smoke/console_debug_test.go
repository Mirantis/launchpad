package smoke_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// dumpConsoleOutputOnFailure fetches and logs the EC2 console output for
// every instance belonging to this smoke test's stack. Register it with
// `defer` *after* `defer terraform.Destroy(...)` in runSmokeTest -- defers
// run LIFO, so this runs first and captures boot/user-data diagnostics
// before the instances are torn down.
//
// Most useful for a stuck WinRM/user-data script (see PRODENG-3471): once
// terraform.Destroy runs, the instance and its logs are gone for good, and a
// bare WinRM "401" gives no way to tell whether user-data ran, errored, or
// never started. This is best-effort -- any AWS API error is logged, never
// fails the test, and only runs when the test already failed so passing
// runs stay quiet.
func dumpConsoleOutputOnFailure(t *testing.T, stackName string) {
	t.Helper()
	if !t.Failed() {
		return
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		t.Logf("DEBUG: console output capture skipped: could not load AWS config: %v", err)
		return
	}
	client := ec2.NewFromConfig(cfg)

	out, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("tag:stack"), Values: []string{stackName}},
			{Name: aws.String("instance-state-name"), Values: []string{"pending", "running", "stopping", "stopped"}},
		},
	})
	if err != nil {
		t.Logf("DEBUG: console output capture skipped: describe-instances failed: %v", err)
		return
	}

	for _, res := range out.Reservations {
		for _, inst := range res.Instances {
			instanceID := aws.ToString(inst.InstanceId)
			label := instanceID
			for _, tag := range inst.Tags {
				if aws.ToString(tag.Key) == "Name" {
					label = aws.ToString(tag.Value) + " (" + instanceID + ")"
				}
			}

			cOut, err := client.GetConsoleOutput(ctx, &ec2.GetConsoleOutputInput{InstanceId: inst.InstanceId})
			if err != nil {
				t.Logf("DEBUG: console output for %s unavailable: %v", label, err)
				continue
			}
			if cOut.Output == nil || *cOut.Output == "" {
				t.Logf("DEBUG: console output for %s: empty (not yet available from EC2 -- note Windows instances rarely populate this; check agent.log via SSM/RDP instead)", label)
				continue
			}
			decoded, err := base64.StdEncoding.DecodeString(*cOut.Output)
			if err != nil {
				t.Logf("DEBUG: console output for %s: failed to decode: %v", label, err)
				continue
			}
			t.Logf("DEBUG: ===== EC2 console output for %s =====\n%s\n===== end console output for %s =====", label, string(decoded), label)
		}
	}
}
