package cmdbuffer_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Mirantis/launchpad/pkg/util/cmdbuffer"
)

func TestParseTestLogrusTxt(t *testing.T) {
	el1 := cmdbuffer.LogEntry{
		Time:  "2025-01-30T17:33:27Z",
		Level: "info",
		Msg:   "message",
	}

	l1, err := cmdbuffer.LogrusParseText(fmt.Sprintf("time=\"%s\" level=%s msg=\"%s\"\n", el1.Time, el1.Level, el1.Msg))

	if el1 != l1 {
		t.Errorf("line parse failed: %+v != %+v", el1, l1)
	}
	if err != nil {
		t.Errorf("line parse gave unexpected error: %v", err)
	}
}

func TestParseTestNotLogrus(t *testing.T) {
	el1 := cmdbuffer.LogEntry{
		Time:  "2025-01-30T17:33:27Z",
		Level: "info",
		Msg:   "message",
	}

	l1, err := cmdbuffer.LogrusParseText("message")

	if el1.Level != l1.Level {
		t.Errorf("line parse failed level: %+v != %+v", el1, l1)
	}
	if el1.Msg != l1.Msg {
		t.Errorf("line parse failed msg: %+v != %+v", el1, l1)
	}
	if err == nil {
		t.Errorf("line parse did not give expected error")
	}
	if !errors.Is(err, cmdbuffer.ErrNotALogRusLine) {
		t.Errorf("line parse gave the wrong error: %v", err)
	}
}
