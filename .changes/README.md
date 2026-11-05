# Release note fragments

One file per change that an operator needs to know about before upgrading.
`.github/workflows/release.yml` assembles every fragment in this directory into
the GitHub release body when a tag is pushed.

## When to add one

Add a fragment when the change alters behaviour an existing configuration or
workflow depends on: a validation rule that rejects input that previously
applied, a default that changes, a flag or field that is removed or renamed, or
anything that makes a documented example stop working.

Do not add one for an internal refactor, a test change, or a fix that makes the
product match its existing documentation.

## Format

Name the file after the ticket, `PRODENG-NNNN.md`. Write the note as plain
markdown — one or two sentences, no heading. Fragments are emitted in filename
order.

State what changed, what it breaks, and **why it was done that way**. The reason
is the part that stops an operator trying to work around the change.

```markdown
`metadata.name` now rejects names containing `/`, `\`, `.`, `..` or
leading/trailing whitespace. Configs using such a name stop applying on
upgrade. Rejection was chosen over silent rewriting because a rewritten name
silently relocates Launchpad's internal storage paths.
```

## After a release

Fragments are not deleted automatically. Remove the ones covered by a release
when cutting the next one, so the notes describe that release rather than
accumulating.

## If there are none

The release body says so explicitly:

```
_No release notes were recorded for this release._
```

That line exists so that a release nobody wrote a fragment for is
distinguishable from a release with genuinely nothing to report. An empty
section would read as the latter while meaning the former.
