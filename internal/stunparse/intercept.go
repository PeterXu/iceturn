package stunparse

import (
	"net"
	"time"

	"github.com/pion/logging"
)

// STUNInterceptConn wraps a net.PacketConn to intercept STUN packets
type STUNInterceptConn struct {
	conn          net.PacketConn
	stunHandler   func([]byte, net.Addr) ([]byte, error)
	logger        logging.LeveledLogger
}

// NewSTUNInterceptConn creates a new STUN-intercepting PacketConn wrapper
func NewSTUNInterceptConn(
	conn net.PacketConn,
	stunHandler func([]byte, net.Addr) ([]byte, error),
	logger logging.LeveledLogger,
) *STUNInterceptConn {
	return &STUNInterceptConn{
		conn:        conn,
		stunHandler: stunHandler,
		logger:      logger,
	}
}

// ReadFrom intercepts incoming packets and processes STUN messages
func (c *STUNInterceptConn) ReadFrom(buf []byte) (n int, addr net.Addr, err error) {
	// Read from underlying connection
	n, addr, err = c.conn.ReadFrom(buf)
	if err != nil {
		return n, addr, err
	}

	// Intercept if it looks like a STUN message
	if n > 0 && IsSTUNMessage(buf[:n]) {
		c.logger.Tracef("Intercepted STUN packet from %s (size=%d)", addr, n)

		// Call STUN handler
		processed, err := c.stunHandler(buf[:n], addr)
		if err != nil {
			c.logger.Warnf("STUN handler error: %v", err)
			// Continue processing even if handler fails
		} else if processed != nil {
			// Replace buffer with processed data
			if len(processed) <= len(buf) {
				copy(buf, processed)
				n = len(processed)
			} else {
				c.logger.Warnf("Processed STUN packet too large: %d > %d", len(processed), len(buf))
			}
		}
	}

	return n, addr, nil
}

// WriteTo writes data to the underlying connection
func (c *STUNInterceptConn) WriteTo(buf []byte, addr net.Addr) (n int, err error) {
	return c.conn.WriteTo(buf, addr)
}

// Close closes the underlying connection
func (c *STUNInterceptConn) Close() error {
	return c.conn.Close()
}

// LocalAddr returns the local address of the underlying connection
func (c *STUNInterceptConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// SetDeadline sets the deadline for the underlying connection
func (c *STUNInterceptConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline for the underlying connection
func (c *STUNInterceptConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline for the underlying connection
func (c *STUNInterceptConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}