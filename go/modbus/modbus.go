// Package modbus is a Modbus TCP client (master) — a Go port of the Node
// @digta/modbus library. Read/write coils and registers over TCP.
// No external dependencies.
package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

var exceptionMessages = map[byte]string{
	1:  "Illegal function",
	2:  "Illegal data address",
	3:  "Illegal data value",
	4:  "Server device failure",
	5:  "Acknowledge",
	6:  "Server device busy",
	8:  "Memory parity error",
	10: "Gateway path unavailable",
	11: "Gateway target device failed to respond",
}

// ModbusError is returned when the server replies with an exception.
type ModbusError struct{ Code byte }

func (e *ModbusError) Error() string {
	if msg := exceptionMessages[e.Code]; msg != "" {
		return "modbus: " + msg
	}
	return fmt.Sprintf("modbus: exception code %d", e.Code)
}

// Client is a Modbus TCP master. Requests are serialized, so it is safe to
// share across goroutines.
type Client struct {
	conn    net.Conn
	unitID  byte
	timeout time.Duration
	txID    uint16
	mu      sync.Mutex
}

// Option configures a Client.
type Option func(*Client)

// WithUnitID sets the Modbus unit/slave id (default 1).
func WithUnitID(id byte) Option { return func(c *Client) { c.unitID = id } }

// WithTimeout sets the per-request timeout (default 2s).
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.timeout = d } }

// Connect dials a Modbus TCP server at addr ("host:port").
func Connect(addr string, opts ...Option) (*Client, error) {
	c := &Client{unitID: 1, timeout: 2 * time.Second}
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

// --- reads ---

// ReadCoils reads quantity coils (FC 0x01).
func (c *Client) ReadCoils(address, quantity uint16) ([]bool, error) {
	return c.readBits(0x01, address, quantity, 2000)
}

// ReadDiscreteInputs reads quantity discrete inputs (FC 0x02).
func (c *Client) ReadDiscreteInputs(address, quantity uint16) ([]bool, error) {
	return c.readBits(0x02, address, quantity, 2000)
}

// ReadHoldingRegisters reads quantity holding registers (FC 0x03).
func (c *Client) ReadHoldingRegisters(address, quantity uint16) ([]uint16, error) {
	return c.readRegisters(0x03, address, quantity)
}

// ReadInputRegisters reads quantity input registers (FC 0x04).
func (c *Client) ReadInputRegisters(address, quantity uint16) ([]uint16, error) {
	return c.readRegisters(0x04, address, quantity)
}

// --- writes ---

// WriteSingleCoil writes one coil (FC 0x05).
func (c *Client) WriteSingleCoil(address uint16, value bool) error {
	pdu := []byte{0x05, byte(address >> 8), byte(address), 0x00, 0x00}
	if value {
		pdu[3] = 0xFF
	}
	_, err := c.request(pdu)
	return err
}

// WriteSingleRegister writes one holding register (FC 0x06).
func (c *Client) WriteSingleRegister(address, value uint16) error {
	pdu := []byte{0x06, byte(address >> 8), byte(address), byte(value >> 8), byte(value)}
	_, err := c.request(pdu)
	return err
}

// WriteMultipleCoils writes multiple coils (FC 0x0F).
func (c *Client) WriteMultipleCoils(address uint16, values []bool) error {
	if len(values) < 1 || len(values) > 1968 {
		return fmt.Errorf("modbus: values length must be 1..1968, got %d", len(values))
	}
	byteCount := (len(values) + 7) / 8
	pdu := make([]byte, 6+byteCount)
	pdu[0] = 0x0F
	binary.BigEndian.PutUint16(pdu[1:], address)
	binary.BigEndian.PutUint16(pdu[3:], uint16(len(values)))
	pdu[5] = byte(byteCount)
	for i, v := range values {
		if v {
			pdu[6+i/8] |= 1 << (uint(i) % 8)
		}
	}
	_, err := c.request(pdu)
	return err
}

// WriteMultipleRegisters writes multiple holding registers (FC 0x10).
func (c *Client) WriteMultipleRegisters(address uint16, values []uint16) error {
	if len(values) < 1 || len(values) > 123 {
		return fmt.Errorf("modbus: values length must be 1..123, got %d", len(values))
	}
	pdu := make([]byte, 6+len(values)*2)
	pdu[0] = 0x10
	binary.BigEndian.PutUint16(pdu[1:], address)
	binary.BigEndian.PutUint16(pdu[3:], uint16(len(values)))
	pdu[5] = byte(len(values) * 2)
	for i, v := range values {
		binary.BigEndian.PutUint16(pdu[6+i*2:], v)
	}
	_, err := c.request(pdu)
	return err
}

// --- internals ---

func (c *Client) readBits(fc byte, address, quantity, max uint16) ([]bool, error) {
	if quantity < 1 || quantity > max {
		return nil, fmt.Errorf("modbus: quantity must be 1..%d, got %d", max, quantity)
	}
	resp, err := c.request(pduReadReq(fc, address, quantity))
	if err != nil {
		return nil, err
	}
	out := make([]bool, quantity)
	for i := range out {
		out[i] = resp[2+i/8]&(1<<(uint(i)%8)) != 0
	}
	return out, nil
}

func (c *Client) readRegisters(fc byte, address, quantity uint16) ([]uint16, error) {
	if quantity < 1 || quantity > 125 {
		return nil, fmt.Errorf("modbus: quantity must be 1..125, got %d", quantity)
	}
	resp, err := c.request(pduReadReq(fc, address, quantity))
	if err != nil {
		return nil, err
	}
	byteCount := int(resp[1])
	if len(resp) < 2+byteCount {
		return nil, fmt.Errorf("modbus: short response")
	}
	out := make([]uint16, byteCount/2)
	for i := range out {
		out[i] = binary.BigEndian.Uint16(resp[2+i*2:])
	}
	return out, nil
}

func pduReadReq(fc byte, address, quantity uint16) []byte {
	return []byte{fc, byte(address >> 8), byte(address), byte(quantity >> 8), byte(quantity)}
}

// request sends a PDU wrapped in an MBAP header and returns the response PDU
// (or an *ModbusError for an exception response).
func (c *Client) request(pdu []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil, fmt.Errorf("modbus: not connected")
	}
	c.txID++
	tx := c.txID

	frame := make([]byte, 7+len(pdu))
	binary.BigEndian.PutUint16(frame[0:], tx)
	binary.BigEndian.PutUint16(frame[2:], 0) // protocol id
	binary.BigEndian.PutUint16(frame[4:], uint16(len(pdu)+1))
	frame[6] = c.unitID
	copy(frame[7:], pdu)

	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	if _, err := c.conn.Write(frame); err != nil {
		return nil, err
	}

	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	head := make([]byte, 6)
	if _, err := io.ReadFull(c.conn, head); err != nil {
		return nil, err
	}
	if got := binary.BigEndian.Uint16(head[0:]); got != tx {
		return nil, fmt.Errorf("modbus: transaction id mismatch: sent %d, got %d", tx, got)
	}
	length := binary.BigEndian.Uint16(head[4:])
	if length < 2 {
		return nil, fmt.Errorf("modbus: invalid length %d", length)
	}
	rest := make([]byte, length) // unit id + PDU
	if _, err := io.ReadFull(c.conn, rest); err != nil {
		return nil, err
	}
	respPDU := rest[1:]
	if respPDU[0]&0x80 != 0 {
		return nil, &ModbusError{Code: respPDU[1]}
	}
	return respPDU, nil
}
