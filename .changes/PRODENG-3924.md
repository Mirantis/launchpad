**`launchpad download-launchpad` now verifies the binary it writes.**

The downloaded asset is checked against the digest the release publishes in
its combined `checksums.sha256` asset. The download is constrained to GitHub
release hosts and to https, including across every redirect hop, and a
non-200 response aborts instead of being written to disk. The binary is
staged in a temporary file, hashed as it exists on disk, and renamed into
place only when the digest matches; a release that publishes no checksums
file, or one that does not list the asset, aborts the download rather than
writing an unverified executable. The previous code opened the destination
without `os.O_TRUNC`, so a shorter new binary left the tail of the old one
in place and produced a corrupt executable from a download that reported
success.

**The Windows MCR install script must now be fetched over a transport that
can authenticate the server.** `spec.mcr.installURLWindows` is accepted only
as an `https://` URL or a local file; any other scheme is rejected instead
of being passed to `http.Get`, a redirect off https is refused, and a
non-200 response aborts rather than being stored as the script and uploaded
to the hosts.

No checksum validation is applied to the install script. An earlier revision
of this change added a `spec.mcr.installURLWindowsChecksum` field; it was
dropped at the user's request, because MCR's Windows PowerShell scripts are
signed -- that signature is the integrity anchor and a second digest adds
nothing -- and because `installURLWindows` is deliberately overridable for
operators who mirror or patch the script, which a pinned digest would
conflict with. `MCRConfig` is unchanged by this release.
