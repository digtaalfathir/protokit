# fins (Go)

Omron **FINS/TCP** client — read/write PLC memory areas in word units.
Implements the FINS/TCP handshake and Memory Area Read/Write (the operations the
Node [fins](../../node/fins) service does via the `omron-fins` library). No
external dependencies.

Import: `github.com/digtaalfathir/protokit/go/fins`

## Usage

```go
c, err := fins.Connect("172.19.88.88:9600")
if err != nil {
    log.Fatal(err)
}
defer c.Close()

words, _ := c.ReadWords(fins.DM, 10000, 10) // D10000..D10009 -> []uint16
_ = c.WriteWords(fins.DM, 100, []uint16{1, 2, 3})
```

Areas: `DM`, `CIO`, `WR`, `HR`. PLC errors come back as `*FinsError` (with
`.EndCode`).
