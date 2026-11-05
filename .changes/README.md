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

Name the file after the ticket, `PRODENG-NNNN.md`. Fragments are emitted into
the release body **verbatim**, in filename order, separated by a blank line.
The fragment owns its own markdown: no list marker or heading is added around
it. A trailing newline is not required: the workflow emits the separating
blank line whether or not the file ends with one.

A short note can be a single paragraph. A note that needs more — what breaks,
what to do about it — may run to several paragraphs; start it with a bold
one-line summary so the release body is scannable. Avoid headings (`#`), which
would compete with the `## Release notes` heading and with GitHub's generated
sections below it.

State what changed, what it breaks, and **why it was done that way**. The reason
is the part that stops an operator trying to work around the change.

```markdown
`metadata.name` now rejects names that contain `/` or `\`, names that are
exactly `.` or `..`, and names with leading or trailing whitespace. Dotted
names such as `prod.eu` are unaffected. Configs using a rejected name stop
applying on upgrade. Rejection was chosen over silent rewriting because a
rewritten name silently relocates Launchpad's internal storage paths.
```

State the rule precisely. "Names containing `.`" and "names that are exactly
`.`" are different claims, and the broader one tells operators their working
configuration is about to break when it is not.

```markdown
**Breaking: one-line summary of what changed.**

What now fails, with the exact error text an operator will see.

Why this shape was chosen over the alternative.
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
