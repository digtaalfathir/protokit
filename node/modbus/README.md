# @digta/modbus

Modbus **TCP** client (master) for Node.js. Promise-based, **zero dependencies**
(uses only the built-in `net` module).

Part of [Protokit](../../README.md).

> ⚠️ Writing to the wrong register can damage equipment or hurt someone. Test
> thoroughly against your PLC before writing in production.

## Install

```bash
npm i @digta/modbus
```

## Quick start

```js
const { ModbusTcpClient } = require("@digta/modbus");

const client = new ModbusTcpClient({ host: "192.168.1.10", port: 502, unitId: 1 });

async function main() {
  await client.connect();

  const regs = await client.readHoldingRegisters(0, 10); // number[] (uint16)
  console.log(regs);

  await client.writeSingleRegister(4, 1234);
  await client.writeMultipleRegisters(0, [1, 2, 3]);

  const coils = await client.readCoils(0, 8); // boolean[]
  await client.writeSingleCoil(0, true);

  client.close();
}

main().catch(console.error);
```

## Options

```js
new ModbusTcpClient({
  host,            // required — PLC IP or hostname
  port: 502,       // Modbus TCP port
  unitId: 1,       // slave / unit id
  timeout: 2000,   // per-request (and connect) timeout, ms
});
```

## API

All methods return a `Promise`. Reads resolve with data; writes resolve with
`undefined`. Addresses are 0-based.

| Method | FC | Returns |
|--------|----|---------|
| `connect()` | — | resolves when connected |
| `readCoils(address, quantity)` | 0x01 | `boolean[]` |
| `readDiscreteInputs(address, quantity)` | 0x02 | `boolean[]` |
| `readHoldingRegisters(address, quantity)` | 0x03 | `number[]` (uint16) |
| `readInputRegisters(address, quantity)` | 0x04 | `number[]` (uint16) |
| `writeSingleCoil(address, boolean)` | 0x05 | — |
| `writeSingleRegister(address, value)` | 0x06 | — |
| `writeMultipleCoils(address, boolean[])` | 0x0F | — |
| `writeMultipleRegisters(address, number[])` | 0x10 | — |
| `close()` | — | closes the socket |
| `connected` (getter) | — | `boolean` |

Quantity limits follow the spec: reads ≤ 2000 bits or ≤ 125 registers; writes
≤ 1968 coils or ≤ 123 registers. Out-of-range arguments throw a `RangeError`
before anything is sent.

## Error handling

Protocol exceptions and transport failures reject with a `ModbusError`:

```js
const { ModbusError } = require("@digta/modbus");

try {
  await client.readHoldingRegisters(9999, 1);
} catch (err) {
  if (err instanceof ModbusError) {
    console.error(err.message, err.code); // e.g. "Illegal data address", 2
  }
}
```

`err.code` is the numeric Modbus exception code for protocol errors, or a string
tag for transport issues: `ETIMEDOUT`, `ENOTCONN`, `ECLOSED`, `EMISMATCH`.

## Reading 32-bit / float values

Registers are 16-bit. Combine two for a 32-bit value (word order depends on your
device):

```js
const [hi, lo] = await client.readHoldingRegisters(0, 2);
const int32 = (hi << 16) | lo;
```

## Scope

This package covers **Modbus TCP** (master/client). Modbus RTU/ASCII over a
serial port is intentionally out of scope — it needs a native serial dependency,
which would break the zero-dependency promise. Modbus RTU-over-TCP gateways work
fine with this client.

## License

[MIT](LICENSE) © Rifky Andigta Al-Fathir
