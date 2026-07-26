// Package mcprotocol is a minimal Mitsubishi MELSEC MC protocol client
// (3E frame, binary) for word devices. The wire format follows the Node
// @digta/mcprotocol library so it talks to the same PLCs.
//
// Scope: batch read/write of word devices (D, W, R, ZR) over TCP. Bit devices
// (M, X, Y) are not implemented yet — MC reads them as words, same as the Node
// library does.
package mcprotocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// DeviceCode is a MELSEC word-device area code (binary 3E).
type DeviceCode byte

const (
	D  DeviceCode = 0xA8 // Data register
	W  DeviceCode = 0xB4 // Link register
	R  DeviceCode = 0xAF // File register
	ZR DeviceCode = 0xB0 // File register (serial)
)

// maxPoints is the 3E batch limit for word units.
const maxPoints = 960

// Client is a single MC protocol connection. Safe for sequential use from
// multiple goroutines (requests are serialized).
type Client struct {
	conn    net.Conn
	timeout time.Duration
	mu      sync.Mutex
}

// Option configures a Client.
type Option func(*Client)

// WithTimeout sets the per-request read/write timeout (default 3s).
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// MCError is returned when the PLC replies with a non-zero end code.
type MCError struct{ EndCode uint16 }

func (e *MCError) Error() string {
	return fmt.Sprintf("mc protocol: PLC returned end code 0x%04X", e.EndCode)
}

// Connect dials a PLC over TCP. addr is "host:port" (the port is whatever you
// configured for the MC protocol connection in GX Works, e.g. 5007).
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
	return c, nil
}

// Close closes the connection.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// ReadWords reads count word registers starting at address from a word device.
func (c *Client) ReadWords(dev DeviceCode, address uint32, count uint16) ([]uint16, error) {
	if count < 1 || count > maxPoints {
		return nil, fmt.Errorf("count must be 1..%d, got %d", maxPoints, count)
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.send(buildRequest(0x0401, dev, address, count, nil)); err != nil {
		return nil, err
	}
	data, err := c.recv()
	if err != nil {
		return nil, err
	}
	if len(data) < int(count)*2 {
		return nil, fmt.Errorf("short response: %d bytes for %d words", len(data), count)
	}
	out := make([]uint16, count)
	for i := range out {
		out[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return out, nil
}

// WriteWords writes values to consecutive word registers starting at address.
func (c *Client) WriteWords(dev DeviceCode, address uint32, values []uint16) error {
	if len(values) < 1 || len(values) > maxPoints {
		return fmt.Errorf("values length must be 1..%d, got %d", maxPoints, len(values))
	}
	payload := make([]byte, len(values)*2)
	for i, v := range values {
		binary.LittleEndian.PutUint16(payload[i*2:], v)
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.send(buildRequest(0x1401, dev, address, uint16(len(values)), payload)); err != nil {
		return err
	}
	_, err := c.recv() // write reply carries only the end code
	return err
}

// buildRequest assembles a 3E binary request frame. command is 0x0401 (read)
// or 0x1401 (write). data is the write payload (nil for reads).
func buildRequest(command uint16, dev DeviceCode, address uint32, points uint16, data []byte) []byte {
	b := make([]byte, 21+len(data))
	b[0], b[1] = 0x50, 0x00                                    // subheader
	b[2] = 0x00                                                // network number
	b[3] = 0xFF                                                // PC number
	binary.LittleEndian.PutUint16(b[4:], 0x03FF)               // request destination module I/O
	b[6] = 0x00                                                // multidrop station
	binary.LittleEndian.PutUint16(b[7:], uint16(12+len(data))) // request data length
	binary.LittleEndian.PutUint16(b[9:], 0x0010)               // monitoring timer
	binary.LittleEndian.PutUint16(b[11:], command)             // command
	binary.LittleEndian.PutUint16(b[13:], 0x0000)              // subcommand: word units
	b[15] = byte(address)                                      // device offset (3 bytes LE)
	b[16] = byte(address >> 8)
	b[17] = byte(address >> 16)
	b[18] = byte(dev)                             // device code
	binary.LittleEndian.PutUint16(b[19:], points) // number of device points
	copy(b[21:], data)
	return b
}

// send writes a full request with a deadline.
func (c *Client) send(frame []byte) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := c.conn.Write(frame)
	return err
}

// recv reads one 3E response and returns its data section (after the end code),
// or an *MCError for a non-zero end code.
func (c *Client) recv() ([]byte, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	// Fixed 9-byte prefix: subheader(2) + network/PC/IO/station(5) + length(2).
	header := make([]byte, 9)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return nil, err
	}
	if header[0] != 0xD0 {
		return nil, fmt.Errorf("unexpected response subheader 0x%02X", header[0])
	}
	length := binary.LittleEndian.Uint16(header[7:]) // end code + data
	body := make([]byte, length)
	if _, err := io.ReadFull(c.conn, body); err != nil {
		return nil, err
	}
	if len(body) < 2 {
		return nil, fmt.Errorf("response too short: %d bytes", len(body))
	}
	if endCode := binary.LittleEndian.Uint16(body[0:]); endCode != 0 {
		return nil, &MCError{EndCode: endCode}
	}
	return body[2:], nil
}
