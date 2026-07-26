// Package openprotocol is a Go port of the Node @digta/open-protocol library:
// an Atlas Copco Open Protocol MID codec plus an ergonomic TCP client
// (auto-reconnect, heartbeat, tightening callbacks). Zero external deps.
package openprotocol

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- codec ---

// BuildFrame builds a raw MID frame: length header + MID + revision + spare, then
// body and a NUL terminator. Mirrors the Node buildFrame (length includes NUL).
func BuildFrame(mid, revision, body string) []byte {
	total := 20 + len(body) + 1
	header := pad(strconv.Itoa(total), 4) + mid + pad(revision, 3) + strings.Repeat(" ", 9)
	return []byte(header + body + "\x00")
}

// The canned MID builders match the exact byte strings the Node library sends.
func BuildMID0001() []byte { return []byte("00200001001" + strings.Repeat(" ", 9) + "\x00") }
func BuildMID0060() []byte { return []byte("002000600011" + strings.Repeat(" ", 8) + "\x00") }
func BuildMID0062() []byte { return []byte("00200062" + strings.Repeat(" ", 12) + "\x00") }
func BuildMID9999() []byte { return []byte("00209999001" + strings.Repeat(" ", 9) + "\x00") }

// BuildMID0050 builds a Vehicle ID (VIN) download; vin is padded/truncated to 25.
func BuildMID0050(vin string) []byte {
	if len(vin) > 25 {
		vin = vin[:25]
	}
	vin += strings.Repeat(" ", 25-len(vin))
	return []byte("00450050" + strings.Repeat(" ", 12) + vin + "\x00")
}

// Frame is a parsed MID frame.
type Frame struct {
	Raw, Length, MID, Revision, Body string
}

// ParseMID parses an incoming frame (NUL stripped).
func ParseMID(data []byte) Frame {
	c := stripNUL(data)
	return Frame{Raw: c, Length: sub(c, 0, 4), MID: sub(c, 4, 8), Revision: sub(c, 8, 11), Body: sub(c, 20, len(c))}
}

// GetReplyMID extracts the replied MID from a MID 0005/0004 response body.
func GetReplyMID(raw string) string { return sub(raw, 20, 24) }

// TighteningResult is the parsed payload of a MID 0061.
type TighteningResult struct {
	VIN                                             string
	PSet                                            int
	TorqueMin, TorqueMax, TorqueTarget, TorqueValue float64
	AngleMin, AngleMax, AngleTarget, AngleValue     int
	TighteningStatus                                int
}

// ParseMID0061 parses a Last Tightening Result frame.
func ParseMID0061(data []byte) TighteningResult {
	c := stripNUL(data)
	return TighteningResult{
		VIN:              strings.TrimSpace(sub(c, 59, 84)),
		PSet:             atoi(sub(c, 90, 93)),
		TorqueMin:        float64(atoi(sub(c, 116, 122))) / 100,
		TorqueMax:        float64(atoi(sub(c, 124, 130))) / 100,
		TorqueTarget:     float64(atoi(sub(c, 132, 138))) / 100,
		TorqueValue:      float64(atoi(sub(c, 140, 146))) / 100,
		AngleMin:         atoi(sub(c, 148, 153)),
		AngleMax:         atoi(sub(c, 155, 160)),
		AngleTarget:      atoi(sub(c, 162, 167)),
		AngleValue:       atoi(sub(c, 169, 174)),
		TighteningStatus: atoi(sub(c, 107, 108)),
	}
}

func stripNUL(b []byte) string { return strings.ReplaceAll(string(b), "\x00", "") }
func sub(s string, start, end int) string {
	if start >= len(s) {
		return ""
	}
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}
func atoi(s string) int { n, _ := strconv.Atoi(strings.TrimSpace(s)); return n }
func pad(s string, n int) string {
	for len(s) < n {
		s = "0" + s
	}
	return s
}

// --- client ---

// Client is an ergonomic Open Protocol TCP client. Set the callbacks, then
// call Connect. It runs the MID 0001/0002/0060/0005 handshake, keeps a MID 9999
// heartbeat, auto-acknowledges results, and reconnects on drop.
type Client struct {
	addr              string
	heartbeatInterval time.Duration
	reconnectDelay    time.Duration
	autoAck           bool
	autoSubscribe     bool

	// Callbacks (all optional). Invoked from the client's read goroutine.
	OnConnect    func()
	OnReady      func()
	OnTightening func(TighteningResult)
	OnDisconnect func()
	OnError      func(error)

	mu      sync.Mutex
	conn    net.Conn
	writeMu sync.Mutex
	lastVIN string
	closed  bool
}

// Option configures a Client.
type Option func(*Client)

func WithHeartbeatInterval(d time.Duration) Option {
	return func(c *Client) { c.heartbeatInterval = d }
}
func WithReconnectDelay(d time.Duration) Option { return func(c *Client) { c.reconnectDelay = d } }
func WithAutoAck(v bool) Option                 { return func(c *Client) { c.autoAck = v } }
func WithAutoSubscribe(v bool) Option           { return func(c *Client) { c.autoSubscribe = v } }

// New creates a client for addr ("host:port").
func New(addr string, opts ...Option) *Client {
	c := &Client{
		addr:              addr,
		heartbeatInterval: 10 * time.Second,
		reconnectDelay:    time.Second,
		autoAck:           true,
		autoSubscribe:     true,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Connect starts the client (non-blocking); it dials and keeps reconnecting
// until Close is called.
func (c *Client) Connect() {
	go c.loop()
}

// SendVin sends a VIN (MID 0050); it is stored and re-sent after reconnect.
func (c *Client) SendVin(vin string) {
	c.mu.Lock()
	c.lastVIN = vin
	c.mu.Unlock()
	c.send(BuildMID0050(vin))
}

// Close stops the client and its reconnect loop.
func (c *Client) Close() {
	c.mu.Lock()
	c.closed = true
	conn := c.conn
	c.mu.Unlock()
	if conn != nil {
		conn.Close()
	}
}

func (c *Client) loop() {
	for {
		if c.isClosed() {
			return
		}
		err := c.session()
		if c.OnDisconnect != nil {
			c.OnDisconnect()
		}
		if err != nil && !c.isClosed() && c.OnError != nil {
			c.OnError(err)
		}
		if c.isClosed() {
			return
		}
		time.Sleep(c.reconnectDelay)
	}
}

func (c *Client) session() error {
	conn, err := net.DialTimeout("tcp", c.addr, 5*time.Second)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	defer conn.Close()

	if c.OnConnect != nil {
		c.OnConnect()
	}
	if err := c.write(conn, BuildMID0001()); err != nil {
		return err
	}

	hbStop := make(chan struct{})
	var hbOnce sync.Once
	stopHB := func() { hbOnce.Do(func() { close(hbStop) }) }
	defer stopHB()

	r := bufio.NewReader(conn)
	for {
		frame, err := r.ReadBytes(0x00) // Open Protocol frames are NUL-terminated
		if err != nil {
			return err
		}
		f := ParseMID(frame)
		switch f.MID {
		case "0002": // comm start acknowledged -> subscribe
			if c.autoSubscribe {
				c.write(conn, BuildMID0060())
			}
		case "0005": // command accepted
			if GetReplyMID(f.Raw) == "0060" {
				c.startHeartbeat(conn, hbStop)
				if c.OnReady != nil {
					c.OnReady()
				}
				if vin := c.getVIN(); vin != "" {
					c.write(conn, BuildMID0050(vin))
				}
			}
		case "0061": // tightening result
			if c.OnTightening != nil {
				c.OnTightening(ParseMID0061(frame))
			}
			if c.autoAck {
				c.write(conn, BuildMID0062())
			}
		}
	}
}

func (c *Client) startHeartbeat(conn net.Conn, stop <-chan struct{}) {
	go func() {
		t := time.NewTicker(c.heartbeatInterval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if err := c.write(conn, BuildMID9999()); err != nil {
					return
				}
			}
		}
	}()
}

// write serializes writes to a specific connection.
func (c *Client) write(conn net.Conn, frame []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err := conn.Write(frame)
	return err
}

// send writes to the current connection if any.
func (c *Client) send(frame []byte) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn != nil {
		c.write(conn, frame)
	}
}

func (c *Client) isClosed() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.closed }
func (c *Client) getVIN() string { c.mu.Lock(); defer c.mu.Unlock(); return c.lastVIN }
