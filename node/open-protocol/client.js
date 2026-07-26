'use strict';

const net = require('net');
const { EventEmitter } = require('events');
const codec = require('./openProtocol');

const DEFAULTS = {
  heartbeatInterval: 10000, // MID 9999 keep-alive interval, ms
  reconnectDelay: 1000,     // wait before reconnecting, ms
  keepAlive: 10000,         // TCP keepalive probe delay, ms
  idleTimeout: 30000,       // force reconnect after this long with no inbound data, ms
  autoReconnect: true,      // reconnect automatically on drop
  autoSubscribe: true,      // subscribe to last tightening (MID 0060) after connect
  autoAck: true,            // auto-send MID 0062 after each tightening result
};

/**
 * Ergonomic Atlas Copco Open Protocol TCP client.
 *
 * Wraps the low-level codec with the whole connection lifecycle: TCP connect +
 * auto-reconnect, the MID 0001/0002/0060/0005 startup handshake, a MID 9999
 * heartbeat, and acknowledging tightening results (MID 0061 -> 0062). You just
 * listen for events and call sendVin().
 *
 * Events:
 *   'connect'      - TCP socket connected (handshake not finished yet)
 *   'ready'        - comm started + subscribed; heartbeat running (fully usable)
 *   'tightening'   - (result, ack) parsed MID 0061 result; ack() sends MID 0062
 *   'heartbeat'    - MID 9999 reply received
 *   'commandError' - ({ mid, code }) MID 0004 from the controller
 *   'disconnect'   - socket closed (auto-reconnect follows unless closed)
 *   'reconnecting' - (delayMs) about to reconnect
 *   'message'      - (packet) any MID not handled above
 *   'error'        - (err) socket error  (attach a listener!)
 *   'close'        - close() was called; no more reconnects
 */
class OpenProtocolClient extends EventEmitter {
  constructor(options = {}) {
    super();
    this.options = Object.assign({}, DEFAULTS, options);
    this.host = options.host || null;
    this.port = options.port || null;
    this._socket = null;
    this._heartbeatTimer = null;
    this._reconnectTimer = null;
    this._lastVin = null;
    this._closed = false;
  }

  /** Connect (or reconnect) to the controller. */
  connect(host = this.host, port = this.port) {
    if (host == null || port == null) {
      throw new Error('OpenProtocolClient.connect: host and port are required');
    }
    this.host = host;
    this.port = port;
    this._closed = false;
    this._open();
    return this;
  }

  /** Send a VIN (MID 0050). Stored and re-sent automatically after reconnect. */
  sendVin(vin) {
    this._lastVin = vin;
    return this._write(codec.buildMID0050(vin));
  }

  /** Manually acknowledge the last tightening result (MID 0062). */
  ack() {
    return this._write(codec.buildMID0062());
  }

  /** Close the connection and stop reconnecting. */
  close() {
    this._closed = true;
    clearTimeout(this._reconnectTimer);
    this._reconnectTimer = null;
    this._stopHeartbeat();
    this._teardownSocket();
    this.emit('close');
    return this;
  }

  /** True once the startup handshake finished and the heartbeat is running. */
  get ready() {
    return this._heartbeatTimer != null;
  }

  // --- internals ---

  _open() {
    this._teardownSocket();

    const socket = new net.Socket();
    this._socket = socket;

    socket.setKeepAlive(true, this.options.keepAlive);
    socket.setTimeout(this.options.idleTimeout);

    socket.connect(this.port, this.host, () => {
      this.emit('connect');
      socket.write(codec.buildMID0001()); // Communication Start
    });

    socket.on('data', (data) => this._onData(data));
    socket.on('timeout', () => socket.destroy()); // idle -> close -> reconnect
    socket.on('error', (err) => this.emit('error', err));
    socket.on('close', () => {
      this._stopHeartbeat();
      this.emit('disconnect');
      if (this.options.autoReconnect && !this._closed) {
        this.emit('reconnecting', this.options.reconnectDelay);
        this._reconnectTimer = setTimeout(() => this._open(), this.options.reconnectDelay);
      }
    });
  }

  _onData(data) {
    // ponytail: one frame per 'data' event (matches the proven CVI3 usage). If
    // you ever see split/coalesced frames, reassemble using the 4-digit length
    // header at bytes 0-3 before parsing.
    const packet = codec.parseMID(data);

    switch (packet.mid) {
      case '0002': // Communication Start acknowledged -> subscribe
        if (this.options.autoSubscribe) this._write(codec.buildMID0060());
        break;

      case '0005': // Command accepted
        if (codec.getReplyMID(packet.raw) === '0060') {
          this._startHeartbeat();
          this.emit('ready');
          if (this._lastVin != null) this._write(codec.buildMID0050(this._lastVin));
        }
        break;

      case '0004': // Command error
        this.emit('commandError', {
          mid: codec.getReplyMID(packet.raw),
          code: packet.raw.substring(24, 26),
        });
        break;

      case '0061': { // Last tightening result
        const result = codec.parseMID0061(packet.raw);
        let acked = false;
        const ack = () => {
          if (acked) return false;
          acked = true;
          return this._write(codec.buildMID0062());
        };
        this.emit('tightening', result, ack);
        if (this.options.autoAck) ack();
        break;
      }

      case '9999': // Heartbeat reply
        this.emit('heartbeat');
        break;

      default:
        this.emit('message', packet);
    }
  }

  _write(buf) {
    if (this._socket && this._socket.writable) {
      this._socket.write(buf);
      return true;
    }
    return false;
  }

  _startHeartbeat() {
    this._stopHeartbeat();
    this._heartbeatTimer = setInterval(() => {
      this._write(codec.buildMID9999());
    }, this.options.heartbeatInterval);
  }

  _stopHeartbeat() {
    if (this._heartbeatTimer) {
      clearInterval(this._heartbeatTimer);
      this._heartbeatTimer = null;
    }
  }

  _teardownSocket() {
    const s = this._socket;
    if (!s) return;
    s.removeAllListeners();
    s.on('error', () => {}); // swallow late errors on the discarded socket
    s.destroy();
    this._socket = null;
  }
}

module.exports = OpenProtocolClient;
