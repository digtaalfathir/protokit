# open-protocol (Go)

Atlas Copco **Open Protocol** — MID codec + ergonomic TCP client. Go port of the
Node [@digta/open-protocol](../../node/open-protocol). No external dependencies.

Import: `github.com/digtaalfathir/protokit/go/open-protocol` (package `openprotocol`)

## Usage

```go
c := openprotocol.New("192.168.0.10:4545")
c.OnReady = func() { log.Println("connected + subscribed") }
c.OnTightening = func(r openprotocol.TighteningResult) {
    log.Printf("torque=%.2f status=%d vin=%s", r.TorqueValue, r.TighteningStatus, r.VIN)
}
c.OnError = func(err error) { log.Println(err) }

c.Connect()
c.SendVin("VIN12345") // re-sent automatically after reconnect
```

Handles TCP connect + auto-reconnect, the MID 0001/0002/0060/0005 handshake, a
MID 9999 heartbeat, and auto-ACK of results (MID 0061 → 0062). The codec
functions (`BuildFrame`, `BuildMID*`, `ParseMID`, `ParseMID0061`, `GetReplyMID`)
are exported too.
