package configurer

import (
	"strings"
	"testing"
)

func TestDockerCmdLinuxQuotesArguments(t *testing.T) {
	t.Parallel()

	var c LinuxConfigurer

	for _, tc := range []struct {
		name string
		arg  string
		want string
	}{
		{name: "plain value is unchanged", arg: "abc123", want: "/usr/bin/docker node rm abc123"},
		{name: "command substitution is quoted", arg: "$(touch /tmp/pwned)", want: `/usr/bin/docker node rm '$(touch /tmp/pwned)'`},
		{name: "command separator is quoted", arg: "x; touch /tmp/pwned", want: `/usr/bin/docker node rm 'x; touch /tmp/pwned'`},
		{name: "embedded single quote is escaped", arg: "a'b", want: `/usr/bin/docker node rm 'a'"'"'b'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := c.DockerCmd().Raw("node rm").Arg(tc.arg).String()
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDockerCmdWindowsQuotesArguments(t *testing.T) {
	t.Parallel()

	var c WindowsConfigurer

	for _, tc := range []struct {
		name string
		arg  string
		want string
	}{
		{name: "plain value is quoted", arg: "abc123", want: `docker.exe node rm ^"abc123^"`},
		{name: "command separator is escaped for cmd.exe", arg: "x & calc", want: `docker.exe node rm ^"x ^& calc^"`},
		{name: "variable expansion is escaped", arg: "%PATH%", want: `docker.exe node rm ^"^%PATH^%^"`},
		{name: "embedded double quote is escaped twice", arg: `a"b`, want: `docker.exe node rm ^"a\^"b^"`},
		{name: "trailing backslash does not escape the closing quote", arg: `C:\dir\`, want: `docker.exe node rm ^"C:\dir\\^"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := c.DockerCmd().Raw("node rm").Arg(tc.arg).String()
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestDockerCmdRawIsNotQuoted documents the escape hatch: Raw is for constants
// and pre-assembled flag lists that must keep their own word splitting.
func TestDockerCmdRawIsNotQuoted(t *testing.T) {
	t.Parallel()

	var c LinuxConfigurer

	got := c.DockerCmd().Raw("run").Raw("--rm -v /var/run/docker.sock:/var/run/docker.sock").Arg("repo/image:1.0").String()
	want := "/usr/bin/docker run --rm -v /var/run/docker.sock:/var/run/docker.sock repo/image:1.0"

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDockerCmdSkipsEmptyRaw(t *testing.T) {
	t.Parallel()

	var c LinuxConfigurer

	got := c.DockerCmd().Raw("run").Raw("").Arg("image").String()
	if strings.Contains(got, "  ") {
		t.Fatalf("empty Raw produced a double space: %q", got)
	}
}
