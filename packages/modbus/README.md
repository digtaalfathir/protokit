# @digta/modbus

Modbus TCP/RTU client for industrial PLC communication.

> **Status: placeholder.** The library API is still being built. A working
> Modbus TCP demo is available in [`examples/`](./examples/) today.

## Install

```bash
npm i @digta/modbus
```

## Usage

The packaged API is not published yet. For now, see the reference demo that
reads holding registers from a PLC over Modbus TCP:

```bash
node examples/modbus-tcp-demo.js
```

```js
const ModbusRTU = require("modbus-serial");
const client = new ModbusRTU();

await client.connectTCP("192.168.250.1", { port: 502 });
client.setID(1);
const { data } = await client.readHoldingRegisters(0, 10);
console.log(data);
```

Part of [Protokit](../../README.md).
