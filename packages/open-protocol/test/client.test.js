'use strict';

// Drives OpenProtocolClient against a fake Open Protocol controller over
// localhost TCP (no real tool). Verifies the startup handshake, tightening
// event + auto-ACK, and the heartbeat.
const assert = require('assert');
const net = require('net');
const codec = require('../openProtocol');
const { OpenProtocolClient } = require('../index');

const received = [];
let heartbeats = 0;
let acked = false;

const server = net.createServer((sock) => {
  sock.on('data', (data) => {
    const { mid } = codec.parseMID(data);
    received.push(mid);
    if (mid === '0001') {
      sock.write(codec.buildFrame('0002')); // Communication Start ack
    } else if (mid === '0060') {
      sock.write(codec.buildFrame('0005', '001', '0060')); // subscribe accepted
      setTimeout(() => sock.write(codec.buildFrame('0061', '001', ' '.repeat(160))), 20); // a result
    } else if (mid === '0062') {
      acked = true;
    } else if (mid === '9999') {
      heartbeats++;
    }
  });
  sock.on('error', () => {});
});

server.listen(0, '127.0.0.1', () => {
  const port = server.address().port;
  const tool = new OpenProtocolClient({ host: '127.0.0.1', port, heartbeatInterval: 40 });

  let ready = false;
  let result = null;
  tool.on('ready', () => { ready = true; });
  tool.on('tightening', (r) => { result = r; });
  tool.on('error', () => {}); // required: EventEmitter throws on unhandled 'error'
  tool.connect();

  setTimeout(() => {
    tool.close();
    server.close();

    assert.ok(received.includes('0001'), 'client sent MID 0001 (start)');
    assert.ok(received.includes('0060'), 'client subscribed with MID 0060');
    assert.ok(ready, "'ready' emitted after subscribe accepted");
    assert.ok(result && typeof result === 'object', "'tightening' emitted with a result object");
    assert.ok(acked, 'client auto-ACKed the result with MID 0062');
    assert.ok(heartbeats >= 1, 'client sent at least one MID 9999 heartbeat');
    console.log('open-protocol client: all checks passed');
  }, 250);
});
