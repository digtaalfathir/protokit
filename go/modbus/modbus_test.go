package modbus

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
)

// fakeServer plays a minimal Modbus TCP server: holding/input registers hold
// (1000 + address + i); reading holding registers at address 40 returns
// exception 2; coils alternate true/false; writes are acknowledged.
func fakeServer(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serve(conn)
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

func serve(conn net.Conn) {
	defer conn.Close()
	for {
		head := make([]byte, 6)
		if _, err := io.ReadFull(conn, head); err != nil {
			return
		}
		rest := make([]byte, binary.BigEndian.Uint16(head[4:]))
		if _, err := io.ReadFull(conn, rest); err != nil {
			return
		}
		unit, pdu := rest[0], rest[1:]
		fc := pdu[0]
		addr := binary.BigEndian.Uint16(pdu[1:])

		var resp []byte
		switch fc {
		case 0x03, 0x04:
			if fc == 0x03 && addr == 40 {
				resp = []byte{fc | 0x80, 0x02}
				break
			}
			n := binary.BigEndian.Uint16(pdu[3:])
			resp = make([]byte, 2+n*2)
			resp[0], resp[1] = fc, byte(n*2)
			for i := uint16(0); i < n; i++ {
				binary.BigEndian.PutUint16(resp[2+i*2:], 1000+addr+i)
			}
		case 0x01, 0x02:
			n := binary.BigEndian.Uint16(pdu[3:])
			bc := (n + 7) / 8
			resp = make([]byte, 2+bc)
			resp[0], resp[1] = fc, byte(bc)
			for i := uint16(0); i < n; i++ {
				if i%2 == 0 {
					resp[2+i/8] |= 1 << (i % 8)
				}
			}
		case 0x05, 0x06:
			resp = pdu[:5] // echo
		case 0x0F, 0x10:
			resp = []byte{fc, pdu[1], pdu[2], pdu[3], pdu[4]} // echo addr + quantity
		default:
			resp = []byte{fc | 0x80, 0x01}
		}

		out := make([]byte, 7+len(resp))
		copy(out[0:], head[0:4]) // txid + protocol
		binary.BigEndian.PutUint16(out[4:], uint16(len(resp)+1))
		out[6] = unit
		copy(out[7:], resp)
		if _, err := conn.Write(out); err != nil {
			return
		}
	}
}

func newClient(t *testing.T) (*Client, func()) {
	addr, stop := fakeServer(t)
	c, err := Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	return c, func() { c.Close(); stop() }
}

func TestReadRegisters(t *testing.T) {
	c, done := newClient(t)
	defer done()

	regs, err := c.ReadHoldingRegisters(0, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint16{1000, 1001, 1002, 1003}
	if len(regs) != len(want) {
		t.Fatalf("got %v want %v", regs, want)
	}
	for i := range want {
		if regs[i] != want[i] {
			t.Fatalf("got %v want %v", regs, want)
		}
	}
}

func TestReadCoils(t *testing.T) {
	c, done := newClient(t)
	defer done()

	coils, err := c.ReadCoils(0, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []bool{true, false, true, false}
	for i := range want {
		if coils[i] != want[i] {
			t.Fatalf("got %v want %v", coils, want)
		}
	}
}

func TestWrites(t *testing.T) {
	c, done := newClient(t)
	defer done()

	if err := c.WriteSingleRegister(4, 1234); err != nil {
		t.Fatal(err)
	}
	if err := c.WriteSingleCoil(1, true); err != nil {
		t.Fatal(err)
	}
	if err := c.WriteMultipleRegisters(0, []uint16{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if err := c.WriteMultipleCoils(0, []bool{true, false, true}); err != nil {
		t.Fatal(err)
	}
}

func TestException(t *testing.T) {
	c, done := newClient(t)
	defer done()

	_, err := c.ReadHoldingRegisters(40, 1)
	var mbErr *ModbusError
	if !errors.As(err, &mbErr) || mbErr.Code != 2 {
		t.Fatalf("want *ModbusError{2}, got %v", err)
	}
}

func TestValidation(t *testing.T) {
	c := &Client{}
	if _, err := c.ReadHoldingRegisters(0, 200); err == nil {
		t.Fatal("expected error for quantity over limit")
	}
}
