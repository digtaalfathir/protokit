// Package fins is a minimal Omron FINS/TCP client for reading and writing PLC
// memory areas in word units. It implements the FINS/TCP handshake and the
// Memory Area Read/Write commands — the operations the Node fins service uses
// via the omron-fins library. No external dependencies.
package fins

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// AreaCode is a FINS memory area code (word units).
type AreaCode byte

const (
	DM  AreaCode = 0x82 // Data Memory (D)
	CIO AreaCode = 0xB0 // Core I/O (word)
	WR  AreaCode = 0xB1 // Work area (word)
	HR  AreaCode = 0xB2 // Holding area (word)
)

// FinsError is returned when the PLC replies with a non-zero end code.
type FinsError struct{ EndCode uint16 }

func (e *FinsError) Error() string {
	return fmt.Sprintf("fins: PLC returned end code 0x%04X", e.EndCode)
}

// Client is a single FINS/TCP connection. Requests are serialized.
type Client struct {
	conn       net.Conn
	timeout    time.Duration
	clientNode byte
	serverNode byte
	sid        byte
	mu         sync.Mutex
}

// Option configures a Client.
type Option func(*Client)

// WithTimeout sets the per-request timeout (default 3s).
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.timeout = d } }

// Connect dials the PLC over TCP and runs the FINS/TCP node-address handshake.
func Connect(addr string, opts ...Option) (*Client, error) {
	c := &Client{timeout: 3 * time.Second}
	for _, o := range opts {
		o(c)
	}
	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	if err := c.handshake(); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

// Close closes the connection.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// ReadWords reads count words from a memory area starting at address.
func (c *Client) ReadWords(area AreaCode, address, count uint16) ([]uint16, error) {
	if count < 1 {
		return nil, fmt.Errorf("fins: count must be >= 1")
	}
	cmd := []byte{
		0x01, 0x01, // MRC, SRC: Memory Area Read
		byte(area),
		byte(address >> 8), byte(address), 0x00, // address (2) + bit (1)
		byte(count >> 8), byte(count),
	}
	data, err := c.command(cmd)
	if err != nil {
		return nil, err
	}
	if len(data) < int(count)*2 {
		return nil, fmt.Errorf("fins: short data: %d bytes for %d words", len(data), count)
	}
	out := make([]uint16, count)
	for i := range out {
		out[i] = binary.BigEndian.Uint16(data[i*2:])
	}
	return out, nil
}

// WriteWords writes values to consecutive words starting at address.
func (c *Client) WriteWords(area AreaCode, address uint16, values []uint16) error {
	if len(values) < 1 {
		return fmt.Errorf("fins: values must not be empty")
	}
	cmd := make([]byte, 8+len(values)*2)
	cmd[0], cmd[1] = 0x01, 0x02 // MRC, SRC: Memory Area Write
	cmd[2] = byte(area)
	cmd[3], cmd[4], cmd[5] = byte(address>>8), byte(address), 0x00
	binary.BigEndian.PutUint16(cmd[6:], uint16(len(values)))
	for i, v := range values {
		binary.BigEndian.PutUint16(cmd[8+i*2:], v)
	}
	_, err := c.command(cmd)
	return err
}

// --- internals ---

func (c *Client) handshake() error {
	// FINS/TCP command 0: node-address-data-send. Client node 0 = auto-assign.
	if err := c.writeTCP(0, []byte{0, 0, 0, 0}); err != nil {
		return err
	}
	body, err := c.readTCP()
	if err != nil {
		return err
	}
	if len(body) < 16 {
		return fmt.Errorf("fins: short handshake response")
	}
	if cmd := binary.BigEndian.Uint32(body[0:]); cmd != 1 {
		return fmt.Errorf("fins: unexpected handshake command %d", cmd)
	}
	if ec := binary.BigEndian.Uint32(body[4:]); ec != 0 {
		return fmt.Errorf("fins: handshake error 0x%08X", ec)
	}
	c.clientNode = byte(binary.BigEndian.Uint32(body[8:]))
	c.serverNode = byte(binary.BigEndian.Uint32(body[12:]))
	return nil
}

// command sends a FINS command (MRC/SRC/params, without the FINS header) and
// returns the response data (after the end code).
func (c *Client) command(finsCmd []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil, fmt.Errorf("fins: not connected")
	}
	c.sid++
	header := []byte{0x80, 0x00, 0x02, 0x00, c.serverNode, 0x00, 0x00, c.clientNode, 0x00, c.sid}
	if err := c.writeTCP(2, append(header, finsCmd...)); err != nil {
		return nil, err
	}
	body, err := c.readTCP()
	if err != nil {
		return nil, err
	}
	if len(body) < 8 {
		return nil, fmt.Errorf("fins: short response")
	}
	resp := body[8:] // FINS frame: header(10) + MRC + SRC + endCode(2) + data
	if len(resp) < 14 {
		return nil, fmt.Errorf("fins: truncated FINS response")
	}
	if end := binary.BigEndian.Uint16(resp[12:]); end != 0 {
		return nil, &FinsError{EndCode: end}
	}
	return resp[14:], nil
}

// writeTCP wraps a payload in a FINS/TCP frame and sends it.
func (c *Client) writeTCP(command uint32, payload []byte) error {
	total := 8 + len(payload) // command(4) + errorCode(4) + payload
	frame := make([]byte, 8+total)
	copy(frame[0:], "FINS")
	binary.BigEndian.PutUint32(frame[4:], uint32(total))
	binary.BigEndian.PutUint32(frame[8:], command)
	binary.BigEndian.PutUint32(frame[12:], 0) // error code
	copy(frame[16:], payload)
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := c.conn.Write(frame)
	return err
}

// readTCP reads one FINS/TCP frame and returns its body (command + errorCode +
// payload — i.e. everything after the length field).
func (c *Client) readTCP() ([]byte, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	head := make([]byte, 8)
	if _, err := io.ReadFull(c.conn, head); err != nil {
		return nil, err
	}
	if string(head[0:4]) != "FINS" {
		return nil, fmt.Errorf("fins: bad magic %q", head[0:4])
	}
	body := make([]byte, binary.BigEndian.Uint32(head[4:]))
	if _, err := io.ReadFull(c.conn, body); err != nil {
		return nil, err
	}
	return body, nil
}
