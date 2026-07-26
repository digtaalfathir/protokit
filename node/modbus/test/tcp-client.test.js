'use strict';

// Drives ModbusTcpClient against a fake Modbus TCP server over localhost.
// Covers: MBAP framing, transaction matching, reads/writes, an exception
// response, and a response split across two TCP segments. No hardware.
const assert = require('assert');
const net = require('net');
const { ModbusTcpClient, ModbusError } = require('../index');

// Minimal Modbus TCP server. Registers hold value (1000 + address). Reading
// holding registers at address 40 returns exception 2. Address 10 gets its
// response deliberately split into two writes.
const server = net.createServer((sock) => {
  let buf = Buffer.alloc(0);
  sock.on('error', () => {});
  sock.on('data', (chunk) => {
    buf = Buffer.concat([buf, chunk]);
    while (buf.length >= 7) {
      const total = 6 + buf.readUInt16BE(4);
      if (buf.length < total) break;
      respond(sock, buf.slice(0, total));
      buf = buf.slice(total);
    }
  });
});

function frame(txId, unitId, pdu) {
  const header = Buffer.alloc(7);
  header.writeUInt16BE(txId, 0);
  header.writeUInt16BE(0, 2);
  header.writeUInt16BE(pdu.length + 1, 4);
  header.writeUInt8(unitId, 6);
  return Buffer.concat([header, pdu]);
}

function respond(sock, req) {
  const txId = req.readUInt16BE(0);
  const unitId = req.readUInt8(6);
  const fc = req.readUInt8(7);
  const addr = req.readUInt16BE(8);
  let pdu;

  if (fc === 0x03 || fc === 0x04) {
    if (fc === 0x03 && addr === 40) {
      pdu = Buffer.from([fc | 0x80, 0x02]); // exception: illegal data address
    } else {
      const count = req.readUInt16BE(10);
      pdu = Buffer.alloc(2 + count * 2);
      pdu.writeUInt8(fc, 0);
      pdu.writeUInt8(count * 2, 1);
      for (let i = 0; i < count; i++) pdu.writeUInt16BE(1000 + addr + i, 2 + i * 2);
    }
  } else if (fc === 0x01 || fc === 0x02) {
    const count = req.readUInt16BE(10);
    const bc = Math.ceil(count / 8);
    pdu = Buffer.alloc(2 + bc);
    pdu.writeUInt8(fc, 0);
    pdu.writeUInt8(bc, 1);
    for (let i = 0; i < count; i++) if (i % 2 === 0) pdu[2 + (i >> 3)] |= 1 << (i & 7);
  } else if (fc === 0x05 || fc === 0x06) {
    pdu = req.slice(7, 12); // echo function code + address + value
  } else if (fc === 0x0f || fc === 0x10) {
    pdu = Buffer.alloc(5);
    pdu.writeUInt8(fc, 0);
    pdu.writeUInt16BE(addr, 1);
    pdu.writeUInt16BE(req.readUInt16BE(10), 3); // echo quantity
  } else {
    pdu = Buffer.from([fc | 0x80, 0x01]);
  }

  const out = frame(txId, unitId, pdu);
  if (fc === 0x03 && addr === 10) {
    sock.write(out.slice(0, 8));                       // split the frame...
    setTimeout(() => sock.write(out.slice(8)), 15);    // ...across two segments
  } else {
    sock.write(out);
  }
}

async function main() {
  const port = await new Promise((res) => server.listen(0, '127.0.0.1', () => res(server.address().port)));
  const client = new ModbusTcpClient({ host: '127.0.0.1', port, unitId: 1 });
  await client.connect();

  assert.deepStrictEqual(await client.readHoldingRegisters(0, 4), [1000, 1001, 1002, 1003], 'read holding registers');
  assert.deepStrictEqual(await client.readInputRegisters(5, 2), [1005, 1006], 'read input registers');
  assert.deepStrictEqual(await client.readCoils(0, 4), [true, false, true, false], 'read coils');

  await client.writeSingleRegister(4, 1234);           // resolves on echo
  await client.writeSingleCoil(1, true);
  await client.writeMultipleRegisters(0, [1, 2, 3]);
  await client.writeMultipleCoils(0, [true, false, true]);

  // exception -> rejected ModbusError with numeric code 2
  await assert.rejects(
    () => client.readHoldingRegisters(40, 1),
    (err) => err instanceof ModbusError && err.code === 2,
    'exception response rejects with ModbusError(code=2)');

  // split-frame response still parses correctly
  assert.deepStrictEqual(await client.readHoldingRegisters(10, 2), [1010, 1011], 'split response reassembled');

  // input validation throws before hitting the wire
  assert.throws(() => client.readHoldingRegisters(0, 999), RangeError, 'quantity over limit rejected');

  client.close();
  server.close();
  console.log('modbus tcp client: all checks passed');
}

main().catch((err) => { console.error(err); process.exit(1); });
