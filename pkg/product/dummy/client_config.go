package dummy

import "fmt"

// ClientConfig does nothing
func (p *Dummy) ClientConfig() error {
	return fmt.Errorf("not implemented for dummy")
}
