package phase

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/k0sproject/rig/v2/cmd"
)

// The line the MKE bootstrapper prints when it generates a password, which
// install_mke.go streams to the log and then scrapes. The password begins
// with the admin username because "admin" is the username install_mke.go
// itself defaults to, so this is the default configuration, not an exotic
// one.
const (
	adminUser   = "admin"
	adminPass   = "admin!s3cr3t"
	bootstrapIn = `msg="Generated random admin password: admin!s3cr3t"`
	// bootstrapWant is the same line with the whole password masked,
	// leaving no tail.
	bootstrapWant = `msg="Generated random [REDACTED] password: [REDACTED]"`
)

// streamRedact pushes s through the writer rig wraps the streamed command
// output in (ExecOptions.logWriter uses RedactWriter), closing it so the
// partial-match buffer is flushed.
func streamRedact(opts *cmd.ExecOptions, s string) string {
	var buf bytes.Buffer
	writer := opts.RedactWriter(&buf)
	if _, err := io.WriteString(writer, s); err != nil {
		panic(err)
	}
	if closer, ok := writer.(io.Closer); ok {
		_ = closer.Close()
	}
	return buf.String()
}

// TestRedactOptionsMasksPasswordContainingUsername pins the ordering
// property through rig's real cmd.Build, so the actual mask and the actual
// sequential replacement apply. Registering the username before the password
// -- which is the order the credentials are named in -- leaves
// "[REDACTED]!s3cr3t" in the log, because the username pass rewrites the
// password's own text before the password pass can match it.
func TestRedactOptionsMasksPasswordContainingUsername(t *testing.T) {
	// Named in the order a caller naturally writes them: username first.
	opts := cmd.Build(redactOptions(adminUser, adminPass)...)

	surfaces := map[string]func() string{
		"RedactString": func() string { return opts.RedactString(bootstrapIn) },
		"Redacter":     func() string { return opts.Redacter().Redact(bootstrapIn) },
		"RedactWriter": func() string { return streamRedact(opts, bootstrapIn) },
	}

	for name, redact := range surfaces {
		t.Run(name, func(t *testing.T) {
			got := redact()
			if strings.Contains(got, "s3cr3t") {
				t.Errorf("password tail survived redaction:\n got %q\nwant %q", got, bootstrapWant)
			}
			if got != bootstrapWant {
				t.Errorf("redacted output mismatch:\n got %q\nwant %q", got, bootstrapWant)
			}
		})
	}
}

// TestRedactOptionsOrderIndependent checks the helper does not merely happen
// to be called with the longer value first.
func TestRedactOptionsOrderIndependent(t *testing.T) {
	userFirst := cmd.Build(redactOptions(adminUser, adminPass)...).RedactString(bootstrapIn)
	passFirst := cmd.Build(redactOptions(adminPass, adminUser)...).RedactString(bootstrapIn)
	if userFirst != passFirst {
		t.Errorf("argument order changed the result:\n user first %q\n pass first %q", userFirst, passFirst)
	}
}

// TestRedactOptionsSkipsEmpty pins the invariant that an unset credential
// registers no pass. An empty match means every position, which
// RedactString expands into a mask between every character of the output;
// the admin username and password are both routinely unset.
func TestRedactOptionsSkipsEmpty(t *testing.T) {
	if got := redactOptions("", ""); len(got) != 0 {
		t.Errorf("unset credentials registered %d redactions, want 0", len(got))
	}

	plain := "msg=\"nothing secret here\""
	if got := cmd.Build(redactOptions("", adminPass)...).RedactString(plain); got != plain {
		t.Errorf("an unset credential corrupted the output:\n got %q\nwant %q", got, plain)
	}
}

// TestRedactOptionsDeduplicates covers a username that equals the password,
// which would otherwise register the same pass twice.
func TestRedactOptionsDeduplicates(t *testing.T) {
	if got := redactOptions(adminPass, adminPass); len(got) != 1 {
		t.Errorf("duplicate credential registered %d redactions, want 1", len(got))
	}
}
