package phase

import (
	"fmt"
	"testing"

	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/k0sproject/rig/exec"
	"github.com/stretchr/testify/require"
)

type testcfg struct {
	Spec *testspec
}

type testspec struct {
	Hosts []*testhost
}

type testhost struct {
	Hooks common.Hooks

	Cmds []string
}

func (t *testhost) String() string {
	return "foo"
}

func (t *testhost) Exec(cmd string, opts ...exec.Option) error {
	return nil
}

func (t *testhost) ExecOutput(cmd string, opts ...exec.Option) (string, error) {
	return "", nil
}

func (t *testhost) WriteFileLarge(src, dest string) error {
	return nil
}

func (t *testhost) ExecAll(cmds []string) error {
	if cmds[0] == "error" {
		return fmt.Errorf("test error")
	}
	t.Cmds = append(t.Cmds, cmds...)
	return nil
}

func TestRun(t *testing.T) {
	host := &testhost{
		Hooks: common.Hooks{
			"apply": {
				"before": []string{"echo hello", "ls -al"},
			},
		},
	}

	d := &testcfg{
		Spec: &testspec{
			Hosts: []*testhost{host},
		},
	}
	p := RunHooks{Action: "apply", Stage: "before"}
	require.NoError(t, p.Prepare(d))
	require.NoError(t, p.Run())
	require.Len(t, host.Cmds, 2)
}

func TestRunError(t *testing.T) {
	host := &testhost{
		Hooks: common.Hooks{
			"apply": {
				"before": []string{"error", "ls -al"},
			},
		},
	}

	d := &testcfg{
		Spec: &testspec{
			Hosts: []*testhost{host},
		},
	}
	host.Cmds = []string{}
	p := RunHooks{Action: "apply", Stage: "before"}
	require.NoError(t, p.Prepare(d))
	require.Error(t, p.Run(), "test error")
	require.Len(t, host.Cmds, 0)
}

func TestTitle(t *testing.T) {
	p := RunHooks{Action: "apply", Stage: "before"}
	require.Equal(t, "Run Before Apply Hooks", p.Title())
}
