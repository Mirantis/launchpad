**Quoted values in `installFlags` and `upgradeFlags` are now decoded as a shell decodes them.**

A flag value is read with POSIX shell rules instead of Go string rules. This matters most for credentials written in a flag rather than in `spec.mke.adminPassword`:

| written in a flag list | before | now |
| --- | --- | --- |
| `--admin-password="C:\temp"` | `C:<TAB>emp` | `C:\temp` |
| `--admin-password="a\b"` | `a<BACKSPACE>` | `a\b` |
| `--ucp-password='p@ss word'` | `'p@ss word'`, quotes included | `p@ss word` |

The MSR bootstrapper read the same configured password two different ways: the installer received the value the shell decoder produced, while destroy, join and remove received the value Go's `strconv.Unquote` produced. A cluster installed with a password containing a backslash could not be destroyed or joined, and the failure surfaced as an authentication error against a cluster that had been working.

Values are still never expanded: `$VAR`, `` `cmd` `` and `$(cmd)` are passed through as written.

If a flag value was previously written with quotes that were expected to survive into the value itself, remove them or escape them: `--san='host'` now yields `host`, and an unquoted backslash is consumed as an escape, so a literal backslash needs `'a\b'` or `"a\b"`.
