# @digta/open-protocol

**Atlas Copco Open Protocol** for Node.js: a MID frame codec **plus** an
ergonomic TCP client. **Zero dependencies.**

> This is the industrial tightening-tool protocol (torque controllers /
> nutrunners), **not** Google Protocol Buffers.

Part of [Protokit](../../README.md).

## Install

```bash
npm i @digta/open-protocol
```

## Quick start — `OpenProtocolClient` (recommended)

You give it a host/port; it handles the rest. TCP connect + **auto-reconnect**,
the MID `0001/0002/0060/0005` startup handshake, a **MID 9999 heartbeat every
10 s**, and **acknowledging** each tightening result (`0061` → `0062`). You just
listen for events and call `sendVin()`.

```js
const { OpenProtocolClient } = require("@digta/open-protocol");

const tool = new OpenProtocolClient({ host: "192.168.0.10", port: 4545 });

tool.on("ready", () => console.log("connected + subscribed"));
tool.on("tightening", (result) => {
  const status = result.tighteningStatus === 1 ? "OK" : "NG";
  console.log(status, "torque:", result.torqueValue, "vin:", result.vin);
});
tool.on("disconnect", () => console.log("dropped — reconnecting…"));
tool.on("error", (err) => console.error(err.message)); // always handle 'error'

tool.connect();
tool.sendVin("VIN12345"); // set the vehicle id (MID 0050); auto-resent after reconnect
```

### Events

| Event | Payload | When |
|-------|---------|------|
| `connect` | — | TCP socket connected (handshake not finished yet) |
| `ready` | — | comm started + subscribed; heartbeat running (fully usable) |
| `tightening` | `(result, ack)` | a tightening result arrived (parsed MID 0061) |
| `heartbeat` | — | MID 9999 reply received |
| `commandError` | `{ mid, code }` | controller rejected a command (MID 0004) |
| `disconnect` | — | socket closed (auto-reconnect follows unless you `close()`) |
| `reconnecting` | `delayMs` | about to reconnect |
| `message` | `packet` | any MID not handled above |
| `error` | `err` | socket error — **attach a listener** |
| `close` | — | `close()` was called; no more reconnects |

### Methods

- `connect(host?, port?)` — connect (defaults to the constructor's host/port).
- `sendVin(vin)` — send MID 0050; stored and re-sent automatically after reconnect.
- `ack()` — manually send MID 0062 (only needed with `autoAck: false`).
- `close()` — disconnect and stop reconnecting.
- `ready` (getter) — `true` once the handshake finished and the heartbeat is running.

### Options (with defaults)

```js
new OpenProtocolClient({
  host, port,
  heartbeatInterval: 10000, // MID 9999 interval, ms
  reconnectDelay: 1000,     // wait before reconnecting, ms
  keepAlive: 10000,         // TCP keepalive probe delay, ms
  idleTimeout: 30000,       // reconnect after no inbound data for this long, ms
  autoReconnect: true,
  autoSubscribe: true,      // subscribe to last tightening (MID 0060) on connect
  autoAck: true,            // auto-send MID 0062 after each result
});
```

Set `autoAck: false` to acknowledge only after you've processed a result
(prevents the tool from advancing if your downstream failed):

```js
const tool = new OpenProtocolClient({ host, port, autoAck: false });
tool.on("tightening", async (result, ack) => {
  if (await forwardSomewhere(result)) ack(); // send MID 0062 only on success
});
```

## Low-level codec

The raw frame builders/parsers are also exported if you need full control:

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

### Drop-in alias for existing code

If your project used a local `require("../utils/openProtocol")`, alias the
package so the bare name keeps working. In the consumer's `package.json`:

```json
{
  "dependencies": {
    "open-protocol": "npm:@digta/open-protocol@^1.0.0"
  }
}
```

```js
const op = require("open-protocol"); // same buildMID*/parseMID* functions
```

## License

[MIT](LICENSE) © Rifky Andigta Al-Fathir
