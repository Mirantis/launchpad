package cmdbuffer_test

import (
	"fmt"
	"testing"

	"github.com/Mirantis/launchpad/pkg/util/cmdbuffer"
)

func TestParseTest(t *testing.T) {
	el1 := cmdbuffer.LogEntry{
		Time:  "2025-01-30T17:33:27Z",
		Level: "info",
		Msg:   "message",
	}

	l1 := cmdbuffer.LogrusParseText(fmt.Sprintf("time=\"%s\" level=%s msg=\"%s\"\n", el1.Time, el1.Level, el1.Msg))

	if el1 != l1 {
		t.Errorf("line parse failed: %+v != %+v", el1, l1)
	}	
}
