package phase

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePhase struct {
	title string
	run   func() error
}

func (f *fakePhase) Title() string { return f.title }
func (f *fakePhase) Run() error    { return f.run() }

func TestManagerRun_NoDeadlineRunsToCompletion(t *testing.T) {
	m := NewManager(struct{}{})
	m.AddPhases(
		&fakePhase{title: "a", run: func() error { return nil }},
		&fakePhase{title: "b", run: func() error { return nil }},
	)

	require.NoError(t, m.Run())
}

func TestManagerRun_DeadlineNotExceededRunsToCompletion(t *testing.T) {
	m := NewManager(struct{}{})
	m.Deadline = time.Second
	m.AddPhases(
		&fakePhase{title: "fast", run: func() error { return nil }},
	)

	require.NoError(t, m.Run())
}

func TestManagerRun_DeadlineExceededNamesTheHungPhase(t *testing.T) {
	m := NewManager(struct{}{})
	m.Deadline = 20 * time.Millisecond
	m.AddPhases(
		&fakePhase{title: "quick", run: func() error { return nil }},
		&fakePhase{title: "hangs-forever", run: func() error {
			select {} // block forever, simulating an unbounded wait
		}},
	)

	err := m.Run()
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "hangs-forever"), "error should name the hung phase, got: %s", err)
	assert.True(t, strings.Contains(err.Error(), "deadline"), "error should mention the deadline, got: %s", err)
}

func TestManagerRun_DeadlineExceededBeforeStartingNextPhase(t *testing.T) {
	m := NewManager(struct{}{})
	m.Deadline = 10 * time.Millisecond
	started := false
	m.AddPhases(
		&fakePhase{title: "slow", run: func() error {
			time.Sleep(50 * time.Millisecond)
			return nil
		}},
		&fakePhase{title: "never-reached", run: func() error {
			started = true
			return nil
		}},
	)

	err := m.Run()
	require.Error(t, err)
	assert.False(t, started, "the second phase must not start once the deadline has already elapsed")
}

func TestManagerRun_PhaseErrorStillPropagatesWithDeadlineSet(t *testing.T) {
	m := NewManager(struct{}{})
	m.Deadline = time.Second
	wantErr := errors.New("boom")
	m.AddPhases(
		&fakePhase{title: "fails", run: func() error { return wantErr }},
	)

	err := m.Run()
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}
