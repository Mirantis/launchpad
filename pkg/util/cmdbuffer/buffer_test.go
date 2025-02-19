package cmdbuffer_test

import (
	"bufio"
	"fmt"
	"testing"
	"time"

	"github.com/Mirantis/launchpad/pkg/util/cmdbuffer"
)

func Test_BufferScanner(t *testing.T) {

	buf := cmdbuffer.NewBuffer()

	go func() {
		buf.Write([]byte("message-0\n"))
		buf.Write([]byte("message-1\n"))
		buf.Write([]byte("message-2\n"))
		time.Sleep(10 * time.Microsecond)
		buf.Write([]byte("mes"))
		time.Sleep(200 * time.Microsecond)
		buf.Write([]byte("sage-3\n"))
		buf.Write([]byte("message-4\n"))
		buf.Write([]byte("message-5\n"))
		time.Sleep(1000 * time.Microsecond)
		buf.Write([]byte("message-6\n"))
		buf.Write([]byte("message-7\n"))
		buf.Write([]byte("message-8\n"))
		time.Sleep(10 * time.Microsecond)
		buf.Write([]byte("message-9\n"))
		buf.Write([]byte("message-10\n"))
		buf.Write([]byte("message-11\n"))
		time.Sleep(200 * time.Microsecond)
		buf.Write([]byte("message-12\n"))
		time.Sleep(10 * time.Microsecond)

		buf.EOF()
	}()

	sc := bufio.NewScanner(buf)

	for i:=0; i<13; i++ {
		expected := fmt.Sprintf("message-%d", i)

		if !sc.Scan() {
			t.Error("unexpected scan failure")
			continue
		}

		got := sc.Text()

		if expected != got {
			t.Errorf("got wrong message: '%s' != '%s'", expected, got)
		}
		
	}

	if sc.Scan() {
		t.Error("buffer scanned more lines than expected")
	}	
	if err := sc.Err(); err != nil {
		t.Error(err)
	}
}
