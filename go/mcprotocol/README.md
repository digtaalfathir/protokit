# mcprotocol (Go)

Minimal Mitsubishi MELSEC **MC protocol** client — **3E frame, binary**, word
devices (D, W, R, ZR) over TCP. The wire format follows the Node
[@digta/mcprotocol](../../node/mcprotocol) library, so it talks to the same PLCs.

Import path: `github.com/digtaalfathir/protokit/go/mcprotocol`

## Usage

```go
c, err := mcprotocol.Connect("192.168.1.10:5007") // port = whatever GX Works is set to
if err != nil {
    log.Fatal(err)
}
defer c.Close()

words, err := c.ReadWords(mcprotocol.D, 0, 10)          // read D0..D9
err = c.WriteWords(mcprotocol.D, 4, []uint16{1234})     // write D4 = 1234
```

A non-zero PLC end code comes back as `*MCError` (with `.EndCode`).

## Runnable demo

```bash
go run ./cmd/mcread -addr 192.168.1.10:5007 -start 0 -count 10
```

`mcread` is a one-shot CLI (reads once and exits). For an always-on poller
supervised by systemd / pm2, see [../README.md](../README.md).

## Scope

- ✅ word read/write (D, W, R, ZR), binary 3E frame
- ⬜ bit devices (M, X, Y) — MC reads them as words; not wired yet
- ⬜ ASCII 3E (the Node fork's ASCII framing fix) — this client uses binary
