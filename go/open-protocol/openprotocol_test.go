package openprotocol

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

func TestCodecRoundtrip(t *testing.T) {
	f := ParseMID(BuildFrame("0050", "001", "BODY"))
	if f.MID != "0050" || f.Revision != "001" || f.Body != "BODY" || f.Length != "0025" {
		t.Fatalf("roundtrip: %+v", f)
	}
	if ParseMID(BuildMID0001()).MID != "0001" {
		t.Fatal("MID0001")
	}
	if ParseMID(BuildMID9999()).MID != "9999" {
		t.Fatal("MID9999")
	}
	if got := GetReplyMID(ParseMID(BuildFrame("0005", "001", "0060")).Raw); got != "0060" {
		t.Fatalf("GetReplyMID = %q", got)
	}
}

// TestClient drives the client against a fake controller that plays the
// handshake and emits one tightening result.
func TestClient(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	acked := make(chan struct{}, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		for {
			frame, err := r.ReadBytes(0x00)
			if err != nil {
				return
			}
			switch ParseMID(frame).MID {
			case "0001":
				conn.Write(BuildFrame("0002", "001", ""))
			case "0060":
				conn.Write(BuildFrame("0005", "001", "0060"))
				conn.Write(BuildFrame("0061", "001", strings.Repeat(" ", 160)))
			case "0062":
				select {
				case acked <- struct{}{}:
				default:
				}
			}
		}
	}()

	ready := make(chan struct{}, 1)
	tight := make(chan TighteningResult, 1)
	c := New(ln.Addr().String(), WithHeartbeatInterval(50*time.Millisecond))
	c.OnReady = func() { trySignal(ready) }
	c.OnTightening = func(r TighteningResult) {
		select {
		case tight <- r:
		default:
		}
	}
	c.OnError = func(error) {}
	c.Connect()
	defer c.Close()

	waitFor(t, ready, "ready")
	select {
	case <-tight:
	case <-time.After(2 * time.Second):
		t.Fatal("no tightening event")
	}
	waitFor(t, acked, "auto-ack (MID 0062)")
}

func trySignal(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func waitFor(t *testing.T, ch chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s", what)
	}
}
