**Executables fetched at runtime are now checked before they are used.**

The Windows MCR install script must be fetched over https or read from a
local file; any other scheme is rejected instead of being passed to
`http.Get`, and a non-200 response aborts rather than being stored as the
script and uploaded to the hosts. A new optional `spec.mcr.installURLWindowsChecksum`
pins the script's SHA-256 for operators who mirror it themselves; a mismatch
aborts the run and there is no override. The field is unset by default
because Mirantis publishes no checksum or signature for the default script:
`https://get.mirantis.com/install.ps1` returns 200 while `install.ps1.sha256`,
`install.ps1.asc`, `install.ps1.sig` and `checksums.sha256` all return 404,
so for the default URL the protection is the https transport.

`launchpad download-launchpad` now verifies the binary against the digest
the release publishes in its combined `checksums.sha256` asset, constrains
the download to GitHub release hosts including across redirects, and checks
the HTTP status. The binary is staged in a temporary file, hashed as it
exists on disk, and renamed into place only when the digest matches. The
previous code opened the destination without `os.O_TRUNC`, so a shorter new
binary left the tail of the old one in place and produced a corrupt
executable from a download that reported success.
