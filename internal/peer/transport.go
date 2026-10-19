package peer

import (
	"crypto/tls"
	"net"
	"time"
)

// Teardown must interrupt blocked IO, not wait for a TLS close-notify write to
// a half-open peer. Session recovery relies on authoritative resynchronization.
func abortConn(c net.Conn) {
	c.SetDeadline(time.Now())
	if tlsConn, ok := c.(*tls.Conn); ok {
		tlsConn.NetConn().Close()
		return
	}
	c.Close()
}
