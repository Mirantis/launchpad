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
