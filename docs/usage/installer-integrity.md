# Integrity of the Windows MCR install script

Launchpad fetches the Windows MCR install script at runtime, uploads it to
every Windows host, and executes it. This describes what is checked before
that happens: how the script is fetched, and whether its signature is
verified on the host.

## Windows MCR install script

`spec.mcr.installURLWindows` names the script. It defaults to
`https://get.mirantis.com/install.ps1`. The bytes are uploaded to every
Windows host and executed, and the same script drives uninstall.

Three checks apply:

- **Scheme.** Only `https://` URLs and local files are accepted. A
  `http://` URL is rejected rather than fetched, because plain HTTP cannot
  authenticate the server that supplies the script. A local path is how an
  air-gapped install supplies its own script; write it as a plain path
  (`C:\mcr\install.ps1`, `/opt/mcr/install.ps1`) rather than a `file://`
  URL, because a `file://` URL carrying a Windows drive letter does not
  parse.
- **Status.** A non-200 response aborts. Previously the body of an error
  response was stored as the install script and uploaded to the hosts.
- **Signature.** The script runs under `-ExecutionPolicy AllSigned`, so
  PowerShell verifies its Authenticode signature on the host before
  executing it and refuses a script whose signature is missing, broken, or
  from a publisher the host does not trust. Launchpad previously ran it
  under `-ExecutionPolicy Bypass`, which disables that check, so the
  signature Mirantis applies was never verified.

The first two checks govern **which bytes arrive**; the third governs
**whether they are authentic**. Neither substitutes for the other: a
signature does not stop launchpad uploading and executing an error page it
was handed instead, and a successful fetch says nothing about who wrote the
script.

### Running an unsigned script

`installURLWindows` is deliberately overridable so operators can point
launchpad at their own mirror or a patched script, and such a script will
not generally carry a signature the host trusts. Set:

```yaml
spec:
  mcr:
    installURLWindows: https://mirror.example.com/install.ps1
    installURLWindowsAllowUnsigned: true
```

That restores `-ExecutionPolicy Bypass` for both the install and uninstall
invocations. It is opt-in and applies to the whole script, so use it only
where the script's provenance is established some other way.

Launchpad does **not** verify a checksum for the script. The signature is
the integrity anchor and a second digest adds nothing on top of it, and a
pinned digest would conflict with the override `installURLWindows` exists to
allow.

Linux hosts are unaffected: they install MCR through their package manager,
which performs its own signature checking, and never fetch this script.
