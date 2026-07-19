'use strict';

const net = require('net');
const { ModbusError, EXCEPTION_MESSAGES } = require('./errors');

const MBAP_HEADER = 7;      // transactionId(2) + protocolId(2) + length(2) + unitId(1)
const PROTOCOL_ID = 0x0000; // always 0 for Modbus
const DEFAULTS = { port: 502, unitId: 1, timeout: 2000 };

// --- small validators (fail fast, before anything hits the wire) ---

function u16(value, what) {
  if (!Number.isInteger(value) || value < 0 || value > 0xffff) {
    throw new RangeError(`${what} must be an integer 0..65535, got ${value}`);
  }
  return value;
}

function qty(value, max, what) {
  if (!Number.isInteger(value) || value < 1 || value > max) {
    throw new RangeError(`${what} must be an integer 1..${max}, got ${value}`);
  }
  return value;
}

/**
 * Modbus TCP client (master). Promise-based, zero dependencies.
 *
 *   const c = new ModbusTcpClient({ host: '192.168.1.10', unitId: 1 });
 *   await c.connect();
 *   const regs = await c.readHoldingRegisters(0, 10);
 *   await c.writeSingleRegister(4, 1234);
 *   c.close();
 *
 * Supported function codes: 0x01 read coils, 0x02 read discrete inputs,
 * 0x03 read holding registers, 0x04 read input registers, 0x05 write single
 * coil, 0x06 write single register, 0x0F write multiple coils, 0x10 write
 * multiple registers.
 */
class ModbusTcpClient {
  constructor(options = {}) {
    if (!options.host) throw new Error('ModbusTcpClient: `host` is required');
    this.host = options.host;
    this.port = options.port != null ? options.port : DEFAULTS.port;
    this.unitId = options.unitId != null ? options.unitId : DEFAULTS.unitId;
    this.timeout = options.timeout != null ? options.timeout : DEFAULTS.timeout;

    this._socket = null;
    this._buffer = Buffer.alloc(0);
    this._txId = 0;
    this._pending = new Map(); // transactionId -> { resolve, reject, timer, fc }
  }

  /** Open the TCP connection. Resolves once connected. */
  connect() {
    return new Promise((resolve, reject) => {
      const socket = new net.Socket();
      this._socket = socket;

      const timer = setTimeout(() => {
        socket.destroy();
        reject(new ModbusError('Connection timed out', 'ETIMEDOUT'));
      }, this.timeout);

      socket.once('error', (err) => { clearTimeout(timer); reject(err); });
      socket.connect(this.port, this.host, () => {
        clearTimeout(timer);
        socket.removeAllListeners('error');
        socket.on('data', (chunk) => this._onData(chunk));
        socket.on('error', (err) => this._failAll(err));
        socket.on('close', () => this._failAll(new ModbusError('Connection closed', 'ECLOSED')));
        resolve();
      });
    });
  }

  /** Close the connection and reject any in-flight requests. */
  close() {
    if (this._socket) {
      this._socket.destroy();
      this._socket = null;
    }
    this._failAll(new ModbusError('Connection closed', 'ECLOSED'));
  }

  get connected() {
    return Boolean(this._socket) && !this._socket.destroyed;
  }

  // --- reads ---

  /** Read `quantity` coils (FC 0x01) → boolean[]. */
  readCoils(address, quantity) {
    return this._readBits(0x01, u16(address, 'address'), qty(quantity, 2000, 'quantity'));
  }

  /** Read `quantity` discrete inputs (FC 0x02) → boolean[]. */
  readDiscreteInputs(address, quantity) {
    return this._readBits(0x02, u16(address, 'address'), qty(quantity, 2000, 'quantity'));
  }

  /** Read `quantity` holding registers (FC 0x03) → number[] (uint16). */
  readHoldingRegisters(address, quantity) {
    return this._readRegisters(0x03, u16(address, 'address'), qty(quantity, 125, 'quantity'));
  }

  /** Read `quantity` input registers (FC 0x04) → number[] (uint16). */
  readInputRegisters(address, quantity) {
    return this._readRegisters(0x04, u16(address, 'address'), qty(quantity, 125, 'quantity'));
  }

  // --- writes ---

  /** Write a single coil (FC 0x05). */
  writeSingleCoil(address, value) {
    const pdu = Buffer.alloc(5);
    pdu.writeUInt8(0x05, 0);
    pdu.writeUInt16BE(u16(address, 'address'), 1);
    pdu.writeUInt16BE(value ? 0xff00 : 0x0000, 3);
    return this._request(pdu).then(() => undefined);
  }

  /** Write a single holding register (FC 0x06). */
  writeSingleRegister(address, value) {
    const pdu = Buffer.alloc(5);
    pdu.writeUInt8(0x06, 0);
    pdu.writeUInt16BE(u16(address, 'address'), 1);
    pdu.writeUInt16BE(u16(value, 'value'), 3);
    return this._request(pdu).then(() => undefined);
  }

  /** Write multiple coils (FC 0x0F) from a boolean[]. */
  writeMultipleCoils(address, values) {
    if (!Array.isArray(values)) throw new TypeError('values must be an array');
    qty(values.length, 1968, 'values.length');
    const byteCount = Math.ceil(values.length / 8);
    const pdu = Buffer.alloc(6 + byteCount);
    pdu.writeUInt8(0x0f, 0);
    pdu.writeUInt16BE(u16(address, 'address'), 1);
    pdu.writeUInt16BE(values.length, 3);
    pdu.writeUInt8(byteCount, 5);
    values.forEach((v, i) => { if (v) pdu[6 + (i >> 3)] |= 1 << (i & 7); });
    return this._request(pdu).then(() => undefined);
  }

  /** Write multiple holding registers (FC 0x10) from a number[] (uint16). */
  writeMultipleRegisters(address, values) {
    if (!Array.isArray(values)) throw new TypeError('values must be an array');
    qty(values.length, 123, 'values.length');
    const pdu = Buffer.alloc(6 + values.length * 2);
    pdu.writeUInt8(0x10, 0);
    pdu.writeUInt16BE(u16(address, 'address'), 1);
    pdu.writeUInt16BE(values.length, 3);
    pdu.writeUInt8(values.length * 2, 5);
    values.forEach((v, i) => pdu.writeUInt16BE(u16(v, `values[${i}]`), 6 + i * 2));
    return this._request(pdu).then(() => undefined);
  }

  // --- internals ---

  _readBits(fc, address, quantity) {
    const pdu = Buffer.alloc(5);
    pdu.writeUInt8(fc, 0);
    pdu.writeUInt16BE(address, 1);
    pdu.writeUInt16BE(quantity, 3);
    return this._request(pdu).then((res) => {
      const bits = [];
      for (let i = 0; i < quantity; i++) {
        bits.push(Boolean((res.readUInt8(2 + (i >> 3)) >> (i & 7)) & 1));
      }
      return bits;
    });
  }

  _readRegisters(fc, address, quantity) {
    const pdu = Buffer.alloc(5);
    pdu.writeUInt8(fc, 0);
    pdu.writeUInt16BE(address, 1);
    pdu.writeUInt16BE(quantity, 3);
    return this._request(pdu).then((res) => {
      const byteCount = res.readUInt8(1);
      const regs = [];
      for (let i = 0; i < byteCount / 2; i++) regs.push(res.readUInt16BE(2 + i * 2));
      return regs;
    });
  }

  _request(requestPdu) {
    return new Promise((resolve, reject) => {
      if (!this.connected) {
        return reject(new ModbusError('Not connected', 'ENOTCONN'));
      }
      const txId = this._nextTxId();
      const header = Buffer.alloc(MBAP_HEADER);
      header.writeUInt16BE(txId, 0);
      header.writeUInt16BE(PROTOCOL_ID, 2);
      header.writeUInt16BE(requestPdu.length + 1, 4); // length = unitId + PDU
      header.writeUInt8(this.unitId, 6);

      const timer = setTimeout(() => {
        this._pending.delete(txId);
        reject(new ModbusError('Request timed out', 'ETIMEDOUT'));
      }, this.timeout);

      this._pending.set(txId, { resolve, reject, timer, fc: requestPdu.readUInt8(0) });
      this._socket.write(Buffer.concat([header, requestPdu]));
    });
  }

  _nextTxId() {
    this._txId = (this._txId + 1) & 0xffff;
    return this._txId;
  }

  _onData(chunk) {
    this._buffer = Buffer.concat([this._buffer, chunk]);
    // A TCP segment may hold a partial frame or several frames. Slice complete
    // MBAP frames using the length field before dispatching.
    while (this._buffer.length >= MBAP_HEADER) {
      const length = this._buffer.readUInt16BE(4); // bytes after the length field
      const total = 6 + length;
      if (this._buffer.length < total) break; // wait for the rest of the frame
      const frame = this._buffer.slice(0, total);
      this._buffer = this._buffer.slice(total);
      this._handleFrame(frame);
    }
  }

  _handleFrame(frame) {
    const txId = frame.readUInt16BE(0);
    const pending = this._pending.get(txId);
    if (!pending) return; // late/unknown response — ignore
    this._pending.delete(txId);
    clearTimeout(pending.timer);

    const pdu = frame.slice(MBAP_HEADER); // function code + data
    const fc = pdu.readUInt8(0);

    if (fc & 0x80) {
      const code = pdu.readUInt8(1);
      return pending.reject(new ModbusError(EXCEPTION_MESSAGES[code] || `Modbus exception ${code}`, code));
    }
    if (fc !== pending.fc) {
      return pending.reject(new ModbusError(`Function code mismatch: expected ${pending.fc}, got ${fc}`, 'EMISMATCH'));
    }
    pending.resolve(pdu);
  }

  _failAll(err) {
    for (const p of this._pending.values()) {
      clearTimeout(p.timer);
      p.reject(err);
    }
    this._pending.clear();
  }
}

module.exports = ModbusTcpClient;
