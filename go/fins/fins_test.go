package fins

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
)

// fakeServer plays a minimal FINS/TCP PLC: the node-address handshake, then
// memory-area reads returning (1000 + i). Reading address 0xFFFF returns end
// code 0x0102; writes succeed.
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
	// Handshake: read command 0, reply command 1 with clientNode=1, serverNode=10.
	if _, err := readFrame(conn); err != nil {
		return
	}
	writeFrame(conn, 1, []byte{0, 0, 0, 1, 0, 0, 0, 10})

	for {
		body, err := readFrame(conn)
		if err != nil {
			return
		}
		if binary.BigEndian.Uint32(body[0:]) != 2 {
			continue
		}
		f := body[8:] // header(10) MRC SRC area addr(2) bit(1) count(2) [data]
		mrc, src := f[10], f[11]
		addr := binary.BigEndian.Uint16(f[13:])
		count := binary.BigEndian.Uint16(f[16:])

		hdr := make([]byte, 10)
		hdr[0] = 0xC0 // response
		switch {
		case src == 0x01 && addr == 0xFFFF:
			writeFrame(conn, 2, append(hdr, mrc, src, 0x01, 0x02)) // end code 0x0102
		case src == 0x01: // read
			data := make([]byte, count*2)
			for i := 0; i < int(count); i++ {
				binary.BigEndian.PutUint16(data[i*2:], uint16(1000+i))
			}
			writeFrame(conn, 2, append(append(hdr, mrc, src, 0x00, 0x00), data...))
		default: // write
			writeFrame(conn, 2, append(hdr, mrc, src, 0x00, 0x00))
		}
	}
}

func readFrame(conn net.Conn) ([]byte, error) {
	head := make([]byte, 8)
	if _, err := io.ReadFull(conn, head); err != nil {
		return nil, err
	}
	body := make([]byte, binary.BigEndian.Uint32(head[4:]))
	_, err := io.ReadFull(conn, body)
	return body, err
}

func writeFrame(conn net.Conn, command uint32, payload []byte) {
	total := 8 + len(payload)
	buf := make([]byte, 8+total)
	copy(buf[0:], "FINS")
	binary.BigEndian.PutUint32(buf[4:], uint32(total))
	binary.BigEndian.PutUint32(buf[8:], command)
	copy(buf[16:], payload)
	_, _ = conn.Write(buf)
}

func TestReadWords(t *testing.T) {
	addr, stop := fakeServer(t)
	defer stop()
	c, err := Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.clientNode != 1 || c.serverNode != 10 {
		t.Fatalf("handshake nodes: client=%d server=%d", c.clientNode, c.serverNode)
	}

	got, err := c.ReadWords(DM, 10000, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint16{1000, 1001, 1002, 1003}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
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

	if err := c.WriteWords(DM, 0, []uint16{1, 2, 3}); err != nil {
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

	_, err = c.ReadWords(DM, 0xFFFF, 1)
	var fe *FinsError
	if !errors.As(err, &fe) || fe.EndCode != 0x0102 {
		t.Fatalf("want *FinsError{0x0102}, got %v", err)
	}
}
