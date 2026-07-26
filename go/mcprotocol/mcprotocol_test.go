package mcprotocol

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
)

// fakeServer plays a minimal MC 3E PLC. Word reads return (1000 + point index);
// reading device offset 0xFFFF returns end code 0xC059; writes succeed.
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
			go serveConn(conn)
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

func serveConn(conn net.Conn) {
	defer conn.Close()
	for {
		header := make([]byte, 9)
		if _, err := io.ReadFull(conn, header); err != nil {
			return
		}
		body := make([]byte, binary.LittleEndian.Uint16(header[7:]))
		if _, err := io.ReadFull(conn, body); err != nil {
			return
		}
		// body: monitorTimer(2) command(2) subcmd(2) offset(3) code(1) points(2) [data]
		command := binary.LittleEndian.Uint16(body[2:])
		offset := uint32(body[6]) | uint32(body[7])<<8 | uint32(body[8])<<16
		points := binary.LittleEndian.Uint16(body[10:])

		var resp []byte
		if offset == 0xFFFF {
			resp = respond(0xC059, nil) // simulated PLC error
		} else if command == 0x0401 {
			data := make([]byte, points*2)
			for i := 0; i < int(points); i++ {
				binary.LittleEndian.PutUint16(data[i*2:], uint16(1000+i))
			}
			resp = respond(0, data)
		} else {
			resp = respond(0, nil) // write ack
		}
		if _, err := conn.Write(resp); err != nil {
			return
		}
	}
}

func respond(endCode uint16, data []byte) []byte {
	b := make([]byte, 11+len(data))
	b[0], b[1] = 0xD0, 0x00
	b[3] = 0xFF
	binary.LittleEndian.PutUint16(b[4:], 0x03FF)
	binary.LittleEndian.PutUint16(b[7:], uint16(2+len(data))) // end code + data
	binary.LittleEndian.PutUint16(b[9:], endCode)
	copy(b[11:], data)
	return b
}

func TestReadWords(t *testing.T) {
	addr, stop := fakeServer(t)
	defer stop()
	c, err := Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	got, err := c.ReadWords(D, 100, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint16{1000, 1001, 1002, 1003}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestWriteWords(t *testing.T) {
	addr, stop := fakeServer(t)
	defer stop()
	c, err := Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if err := c.WriteWords(D, 0, []uint16{1, 2, 3}); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestEndCodeError(t *testing.T) {
	addr, stop := fakeServer(t)
	defer stop()
	c, err := Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	_, err = c.ReadWords(D, 0xFFFF, 1)
	var mcErr *MCError
	if !errors.As(err, &mcErr) || mcErr.EndCode != 0xC059 {
		t.Fatalf("want *MCError{0xC059}, got %v", err)
	}
}

func TestReadWordsValidation(t *testing.T) {
	c := &Client{}
	if _, err := c.ReadWords(D, 0, 0); err == nil {
		t.Fatal("expected error for count=0")
	}
	if _, err := c.ReadWords(D, 0, maxPoints+1); err == nil {
		t.Fatal("expected error for count over limit")
	}
}
