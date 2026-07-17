# @digta/open-protocol

Builder and parser for **Atlas Copco Open Protocol** MID frames (ASCII).
**Zero dependencies.**

> This is the industrial tightening-tool protocol (torque controllers /
> nutrunners), **not** Google Protocol Buffers.

Part of [Protokit](../../README.md).

## Install

```bash
npm i @digta/open-protocol
```

### Drop-in alias for existing code

If your project used a local `require("../utils/openProtocol")`, you can alias
the package so the bare name keeps working. In the consumer's `package.json`:

```json
{
  "dependencies": {
    "open-protocol": "npm:@digta/open-protocol@^1.0.0"
  }
}
```

Then in code:

```js
const op = require("open-protocol");
```

## API

```js
const op = require("@digta/open-protocol");
```

| Function | Purpose |
|----------|---------|
| `buildFrame(mid, revision?, body?)` | Build a raw MID frame (length header + body + NUL) |
| `buildMID0001()` | Communication Start |
| `buildMID0050(vin)` | Vehicle ID Number download (send VIN) |
| `buildMID0060()` | Subscribe to Last Tightening Result |
| `buildMID0062()` | Acknowledge Last Tightening Result |
| `buildMID9999()` | Heartbeat / keep-alive |
| `parseMID(buf)` | Parse an incoming frame → `{ raw, length, mid, revision, body }` |
| `parseMID0061(raw)` | Parse a tightening result (torque, angle, VIN, status, …) |
| `getReplyMID(raw)` | Extract the replied MID from a MID 0005/0004 response |

## Example

```js
const net = require("net");
const op = require("@digta/open-protocol");

const client = net.connect(4545, "192.168.0.10", () => {
  client.write(op.buildMID0001());
});

client.on("data", (data) => {
  const { mid, raw } = op.parseMID(data);
  if (mid === "0002") client.write(op.buildMID0060()); // start accepted → subscribe
  if (mid === "0061") {
    const result = op.parseMID0061(raw);
    console.log(result);
    client.write(op.buildMID0062()); // acknowledge
  }
});
```

## License

[MIT](LICENSE) © Rifky Andigta Al-Fathir
