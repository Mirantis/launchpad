# Getting Started with Mirantis Launchpad

Launchpad is a single-binary CLI for managing MKE and MCR clusters.

## Installation

Download the binary for your platform (Linux, macOS, or Windows) from the [latest GitHub releases](https://github.com/Mirantis/launchpad/releases).

For Linux and macOS (AMD64 and ARM64):
- Download the binary.
- Make it executable: `chmod +x launchpad`.
- Move it to your path: `sudo mv launchpad /usr/local/bin/`.

## Provisioning Compute Nodes

Before using Launchpad, you must provision your compute nodes. Ensure the nodes follow the [Mirantis predeployment documentation](https://docs.mirantis.com/mke/3.8/install/predeployment.html).

### Terraform Helpers
Mirantis provides Terraform modules for various clouds:
- **AWS**: [provision-aws](https://github.com/terraform-mirantis-modules/terraform-mirantis-provision-aws)
- **Azure**: [provision-azure](https://github.com/terraform-mirantis-modules/terraform-mirantis-provision-azure)
- **GCP**: [provision-gcp](https://github.com/terraform-mirantis-modules/terraform-mirantis-provision-gcp)

## Configuration (`launchpad.yaml`)

Create a `launchpad.yaml` to define your cluster. A minimal configuration includes:
- **Hosts**: IP addresses, usernames, and roles (manager, worker).
- **MKE Product**: Specific versions and license information.

For a full reference, see the [public documentation](https://docs.mirantis.com/mke/3.8/launchpad.html).

### Keeping credentials out of `launchpad.yaml`

Do not write passwords into `launchpad.yaml`. The file sits in plaintext on
the disk of whichever machine runs Launchpad, is read by every `apply` and
`reset`, and is easy to paste into a ticket or a chat window by accident.

Launchpad expands environment variables in the configuration file before
parsing it, so reference the credential instead of inlining it:

```yaml
spec:
  hosts:
    - role: manager
      winRM:
        address: 10.0.0.1
        user: Administrator
        password: ${WINRM_PASSWORD}
        useHTTPS: true
  mke:
    adminUsername: admin
    adminPassword: ${MKE_ADMIN_PASSWORD}
```

Supply the values from your shell, a secret manager, or a CI secret store:

```bash
export WINRM_PASSWORD="$(my-secret-tool read winrm/administrator)"
export MKE_ADMIN_PASSWORD="$(my-secret-tool read mke/admin)"
launchpad apply --config launchpad.yaml
```

Both `${VAR}` and `$VAR` are recognised. Note the substitution rules, because
they bite in exactly this situation:

- **An unset variable is an error, not an empty string.** If
  `WINRM_PASSWORD` is not exported, `launchpad apply` fails with a
  substitution error rather than quietly trying to authenticate with an empty
  password. An exported-but-empty variable is accepted as empty.
- **A literal `$` in a value must be escaped as `$$`.** A password of
  `Pa$$word` has to be written `Pa$$$$word`, and a password of `abc$def` has
  to be written `abc$$def` — otherwise `$def` is read as a variable
  reference. This applies to any value in the file, not just passwords.
- **`$` followed by a digit is left alone.** `$5986` is never a valid
  variable name, so it stays literal and needs no escaping.
- **The substituted value is parsed as YAML afterwards.** Substitution
  rewrites the file's text and the result is then re-parsed, so YAML
  metacharacters inside the value are interpreted rather than taken
  literally. A password of `abc #def` is truncated to `abc`, because
  everything from the unquoted `#` is a comment; a password of `abc: def`
  aborts the load with `yaml: line 11: mapping values are not allowed in
  this context`. Quoting the placeholder in your own file does not help:
  the configuration is re-marshalled before substitution, which discards
  that quoting.

A value containing `#`, `:`, `"`, `'`, a leading `{`, `[`, `*`, `&` or `%`,
or a trailing space therefore needs care. Two ways to stay safe:

- Prefer generated passwords restricted to alphanumerics and `-`, `_`, `.`,
  which have no YAML meaning in any position.
- If the value is outside your control, pass it base64-encoded and decode it
  when you set the variable, so the text that reaches the configuration file
  is always inert.

Sourcing the credential from the environment is still better than inlining
it, because the value lives in the process environment rather than on disk —
but it is not a way to smuggle arbitrary bytes past the parser.

### WinRM over TLS

Launchpad warns at config load for every host whose WinRM password is exposed
by the transport it is configured to use. There are two separate failures.

**`winRM.useHTTPS: false` with a password sends it in clear.** Without TLS,
and without `winRM.useNTLM: true`, the password is placed in an HTTP
`Authorization` header as base64 — an encoding, not encryption. Anyone on the
network path reads the password and every command sent to the host. This is
the default transport: `useHTTPS` defaults to `false`.

**`winRM.insecure: true` accepts any certificate the host presents.** The
channel is encrypted but the peer is not authenticated, so an attacker on the
network path can impersonate the host, capture the password and read every
command sent to it. `useHTTPS: true` does not protect you on its own.

Treat either warning as a finding. Set `winRM.useHTTPS: true`, install a
certificate the client trusts, pin it with `winRM.caCertPath`, and leave
`winRM.insecure` unset. Leaving `insecure` set is only defensible on a
trusted network you control, and never with a password you would mind losing.

## Running Launchpad

Use the `apply` command to start the installation:
```bash
launchpad apply --config launchpad.yaml
```

To remove managed products from the cluster:
```bash
launchpad reset --config launchpad.yaml
```
