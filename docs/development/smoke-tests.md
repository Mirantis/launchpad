# Writing Smoke Tests

Smoke tests live in `test/smoke/` and use [Terratest](https://terratest.gruntwork.io/) to provision real AWS infrastructure, run the full Launchpad lifecycle, and tear down — all within a single `go test` invocation.

Read this document before adding a new test. The framework enforces several invariants that must be preserved.

---

## How the framework works

### Infrastructure

All tests share a single Terraform module: `examples/terraform/aws-simple/`. Tests pass variables to it; the module provisions VPCs, subnets, security groups, EC2 instances, an NLB, IAM roles, and SSH key pairs, then outputs a ready-to-use `launchpad_yaml` string.

The Terraform state is ephemeral — it lives in a temp directory for the duration of the test and is never committed.

### Lifecycle

Every test follows this sequence:

```
terraform init + apply   → provision AWS infra
  ↓
config.ProductFromYAML   → parse Terraform's launchpad_yaml output
  ↓
product.Apply(...)       → install (and optionally upgrade) MKE/MCR via Launchpad
  ↓
product.Reset()          → uninstall (best-effort, non-fatal — see below)
  ↓
terraform destroy        → unconditional, runs via defer even on t.Fatal
```

`defer terraform.Destroy` is registered **before** `InitAndApply`, so infrastructure is always cleaned up even when apply or the test itself fails.

### Resource tagging

Every test must tag all AWS resources so they can be tracked and audited independently of Terraform state:

```go
"extra_tags": map[string]string{
    "launchpad-smoke-test":      "true",
    "launchpad-smoke-test-name": cfg.Name,   // e.g. "modern", "legacy", "upgrade"
},
```

---

## Available platforms

Platforms are defined in `test/platforms.go` as `test.Platform` values in the `test.Platforms` map. Each platform knows its Terraform platform key, instance sizing, firewall `user_data`, and how to produce a manager, worker, or MSR nodegroup map.

| Key | Terraform platform | OS |
|---|---|---|
| `Ubuntu20` | `ubuntu_20.04` | Ubuntu 20.04 |
| `Ubuntu22` | `ubuntu_22.04` | Ubuntu 22.04 |
| `Ubuntu24` | `ubuntu_24.04` | Ubuntu 24.04 |
| `Rhel8` | `rhel_8` | RHEL 8 |
| `Rhel9` | `rhel_9` | RHEL 9 |
| `Rocky8` | `rocky_8` | Rocky Linux 8 |
| `Rocky9` | `rocky_9` | Rocky Linux 9 |
| `Sles12` | `sles_12` | SLES 12 |
| `Sles15` | `sles_15` | SLES 15 |
| `Oracle9` | `oracle_9` | Oracle Linux 9 |
| `Centos7` | `centos_7` | CentOS 7 |
| `Windows2019` | `windows_2019` | Windows Server 2019 |
| `Windows2022` | `windows_2022` | Windows Server 2022 |
| `Windows2025` | `windows_2025` | Windows Server 2025 |

Each platform exposes three methods:

```go
test.Platforms["Rhel9"].GetManager()   // m6a.2xlarge, role=manager
test.Platforms["Rhel9"].GetWorker()    // c6a.xlarge,  role=worker
test.Platforms["Rhel9"].GetMSR()       // m6a.2xlarge, role=msr
```

To add a new platform, add an entry to `test/platforms.go`. For platforms not supported by the upstream Terraform module (currently `ubuntu_24.04` and `windows_2025`), also add a local definition in `examples/terraform/aws-simple/platform.tf` under `lib_local_platform_definitions`.

---

## Writing an install/reset test

Use `runSmokeTest` (defined in `test/smoke/smoke_test.go`). Fill in a `smokeConfig` and call it from a `Test*` function.

```go
// test/smoke/my_test.go
package smoke_test

import (
    "testing"
    "github.com/Mirantis/launchpad/test"
)

func TestMyCluster(t *testing.T) {
    runSmokeTest(t, smokeConfig{
        // Name is used in the AWS resource name and launchpad-smoke-test-name tag.
        // Keep it short and lowercase (it forms part of resource names like
        // "smoke-mytest-XXXXX-MngrRhel9-0").
        Name: "mytest",

        // MCR channel, e.g. "stable-29.2", "stable-25.0".
        MCRChannel: "stable-29.2",

        // MKE and MSR versions.
        MKEVersion: "3.9.2",
        MSRVersion: "3.1.18",

        // SSH key algorithm: "ed25519" (default) or "rsa".
        // Use "rsa" only when the cluster includes Windows nodes —
        // RSA is required for AWS Windows password retrieval.
        SSHKeyAlgorithm: "ed25519",

        // Nodegroups: map of unique nodegroup name → platform nodegroup map.
        // Naming convention: prefix with role (Mngr/Wrk/Msr) + platform,
        // e.g. "MngrRhel9", "WrkUbuntu24", "MsrRhel9".
        // Names must be unique within the test.
        Nodegroups: map[string]interface{}{
            "MngrRhel9":    test.Platforms["Rhel9"].GetManager(),
            "WrkUbuntu24":  test.Platforms["Ubuntu24"].GetWorker(),
            "WrkRocky9":    test.Platforms["Rocky9"].GetWorker(),
        },
    })
}
```

`runSmokeTest` handles everything: infra provisioning, password generation, tagging, `defer` destroy, `Apply`, and best-effort `Reset`.

### Windows clusters

Include at least one Linux manager. Pass `SSHKeyAlgorithm: "rsa"` — RSA 4096-bit keys are required for AWS's encrypted Windows password mechanism. The framework auto-detects Windows nodegroups (by `platform` prefix `windows_`) and generates a compliant password via `generateWindowsPassword`.

```go
func TestMyWindowsCluster(t *testing.T) {
    runSmokeTest(t, smokeConfig{
        Name:            "mywindows",
        MCRChannel:      "stable-25.0",
        MKEVersion:      "3.8.8",
        MSRVersion:      "2.9.28",
        SSHKeyAlgorithm: "rsa",   // required for Windows
        Nodegroups: map[string]interface{}{
            "MngrUbuntu24": test.Platforms["Ubuntu24"].GetManager(),
            "WrkWin2022":   test.Platforms["Windows2022"].GetWorker(),
            "WrkWin2025":   test.Platforms["Windows2025"].GetWorker(),
        },
    })
}
```

---

## Writing an upgrade test

Use `runUpgradeTest` (defined in `test/smoke/upgrade_test.go`). It provisions infra once, installs the base versions, then calls `Apply` a second time with mutated versions to trigger the upgrade.

```go
func TestMyUpgrade(t *testing.T) {
    runUpgradeTest(t, upgradeConfig{
        Base: smokeConfig{
            Name:            "myupgrade",
            MCRChannel:      "stable-25.0",
            MKEVersion:      "3.8.8",
            MSRVersion:      "2.9.28",
            SSHKeyAlgorithm: "ed25519",
            Nodegroups: map[string]interface{}{
                "MngrUbuntu22": test.Platforms["Ubuntu22"].GetManager(),
                "WrkRhel8":     test.Platforms["Rhel8"].GetWorker(),
            },
        },
        UpgradeMCRChannel: "stable-29.2",
        UpgradeMKEVersion: "3.9.2",
    })
}
```

`bumpVersions` (in `upgrade_test.go`) handles the YAML mutation between the two `Apply` calls: it unmarshals the Terraform output, updates `spec.mcr.channel` and `spec.mke.version`, and re-marshals — preserving host addresses, SANs, LB names, and all install flags exactly as Terraform generated them.

If you need to also change `spec.msr.version` during upgrade, extend `bumpVersions` or add a new mutator alongside it.

---

## Hooking into CI

### 1. Add a Makefile target

```makefile
.PHONY: smoke-mytest
smoke-mytest:
	go test -count=1 -v ./test/smoke/... -run TestMyCluster -timeout 50m
```

Timeout guidance:
- Install-only tests: **50m**
- Windows tests: **60m** (WinRM setup and Windows image pull are slower)
- Upgrade tests: **90m** (two full apply cycles)
- Multi-hop upgrade tests with image preloading: **200m+** (N apply cycles plus per-node image pulls across all upgrade versions)

### 2. Add a CI job

Add a job to `.github/workflows/smoke-tests.yaml` following the existing pattern:

```yaml
smoke-mytest:
  runs-on: ubuntu-latest
  if: |
    github.event_name == 'push' ||
    contains(github.event.pull_request.labels.*.name, 'smoke-test') ||
    contains(github.event.pull_request.labels.*.name, 'smoke-mytest')
  steps:
    - uses: actions/checkout@v4
    - uses: hashicorp/setup-terraform@v3
    - name: Run my smoke test
      env:
        AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
        AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
      run: make smoke-mytest
```

### 3. Create the PR label

```bash
gh label create smoke-mytest --repo Mirantis/launchpad --color "0075ca"
```

Then add the label to your PR to trigger only that job, or add `smoke-test` to trigger all smoke jobs.

---

## Reset and cleanup behaviour

`product.Reset()` is called after `Apply` but is treated as **best-effort** and non-fatal. The MKE uninstall bootstrapper has a hardcoded node-response timeout (~2 minutes) that can fire on large or mixed-OS clusters before all uninstall agents report back. Because `defer terraform.Destroy` runs unconditionally, no AWS resources are orphaned if Reset fails.

```go
// Pattern used in all smoke tests — do not assert on Reset.
if err = product.Reset(); err != nil {
    t.Logf("WARN: product.Reset() failed (non-fatal): %v", err)
}
```

If `Reset` fails, Launchpad will fall back to a forced swarm dissolution (see `pkg/product/mke/phase/uninstall_mke.go`). Do not change this to a hard assertion without first confirming that `Reset` is reliable for your cluster size and platform.

---

## Checklist for new smoke tests

- [ ] Test function name starts with `Test` and is descriptive (`TestMyCluster`, not `TestIt`).
- [ ] `smokeConfig.Name` is short, lowercase, and unique across all tests.
- [ ] `SSHKeyAlgorithm` is `"rsa"` if any Windows nodegroups are present.
- [ ] `extra_tags` includes `launchpad-smoke-test: true` and `launchpad-smoke-test-name` — handled automatically by `runSmokeTest`/`runUpgradeTest`.
- [ ] `defer terraform.Destroy` is registered before `InitAndApply` — handled automatically by helpers.
- [ ] Makefile target added with an appropriate timeout.
- [ ] CI job added to `.github/workflows/smoke-tests.yaml`.
- [ ] PR label created (`gh label create smoke-<name>`).
- [ ] New platform? Add to `test/platforms.go` and, if needed, `examples/terraform/aws-simple/platform.tf`.
