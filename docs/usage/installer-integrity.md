# Integrity of fetched executables

Launchpad fetches two kinds of executable content at runtime: the Windows
MCR install script, and -- when `launchpad download-launchpad` is run -- a
launchpad release binary. Both are now checked before they are used.

## Windows MCR install script

`spec.mcr.installURLWindows` names the script. It defaults to
`https://get.mirantis.com/install.ps1`. The bytes are uploaded to every
Windows host and executed with `-ExecutionPolicy Bypass`, and the same
script drives uninstall.

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
- **Checksum, when configured.** `spec.mcr.installURLWindowsChecksum` pins
  the SHA-256 of the script, hex encoded. A mismatch aborts the run.

```yaml
spec:
  mcr:
    installURLWindows: https://mirror.example.com/mcr/install.ps1
    installURLWindowsChecksum: 3b7a...  # sha256sum install.ps1
```

`installURLWindowsChecksum` is unset by default, and that is deliberate
rather than an omission: **Mirantis publishes no checksum or signature for
`install.ps1`.** As measured in PRODENG-3924,
`https://get.mirantis.com/install.ps1` returns 200, while
`install.ps1.sha256`, `install.ps1.asc`, `install.ps1.sig` and
`checksums.sha256` on the same host all return 404. There is nothing
upstream to verify the default script against, so for the default URL the
protection is the HTTPS transport. Operators who mirror the script
themselves know its digest and can set the field.

There is no override for a failed check. A pin that can be ignored is worse
than no pin, because the configuration would claim a guarantee the run does
not provide.

Linux hosts are unaffected: they install MCR through their package manager,
which performs its own signature checking, and never fetch this script.

## `launchpad download-launchpad`

The downloaded binary is verified against the digest the release publishes
before it replaces the local file.

- The release's combined `checksums.sha256` asset is fetched and the line
  for the current OS/architecture asset is used. A release that publishes
  no checksums file, or that does not list the asset, aborts the download.
- The asset host must be a GitHub release host, both for the initial URL
  and for every redirect hop.
- The download is written to a temporary file in the destination directory,
  hashed **as it exists on disk**, and renamed into place only if the digest
  matches. A rejected download is removed.

Hashing the file rather than the response matters: the previous code opened
the destination with `os.O_CREATE|os.O_WRONLY` and no `os.O_TRUNC`, so
overwriting a longer existing binary with a shorter one left the tail of the
old file in place. The result was a corrupt executable produced by a
download that reported success, and a digest checked against the response
bytes would still have passed.
