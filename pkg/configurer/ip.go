package configurer

import "net"

// isValidAddress checks whether the given IP address is a valid one.
func isValidAddress(address string) bool {
	return net.ParseIP(address) != nil
}
