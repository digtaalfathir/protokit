'use strict';

// Tests the ASCII 3E TCP frame reassembly fix by driving the REAL on('data')
// handler (attached in onTCPConnect) with a fake socket — no PLC required.
// See CHANGELOG.md for what this guards against.

const assert = require('assert');
const EventEmitter = require('events');
const MC = require('../mcprotocol.js');

// Build a valid ASCII 3E response frame around a data body.
// Header (18 chars): subheader(4) + access route(10) + data-length(4 hex chars).
// data-length = number of ASCII chars that follow the header.
function frame(body) {
  const len = body.length.toString(16).toUpperCase().padStart(4, '0');
  return 'D000' + '00FF03FF00' + len + body; // 4 + 10 + 4 + body.length
}

// Drive the real reassembly handler over a sequence of raw chunks (ascii 3E).
function collectFrames(chunks) {
  const conn = new MC();
  const sock = new EventEmitter();
  sock.remoteAddress = '127.0.0.1';
  sock.remotePort = 0;
  conn.isoclient = sock;
  conn.isAscii = true;
  conn.frame = '3E';

  const got = [];
  conn.onResponse = (f) => got.push(Buffer.from(f).toString('ascii'));
  conn.connectionReset = () => got.push('__RESET__');

  const log = console.log;
  console.log = () => {}; // silence the library's connect log during setup
  try {
    conn.onTCPConnect(); // attaches the reassembly 'data' handler
    chunks.forEach((c) => sock.emit('data', Buffer.from(c, 'ascii')));
  } finally {
    console.log = log;
  }
  return got;
}

const A = frame('00001234'); // dataLen 8  -> 26 chars total
const B = frame('0000');     // dataLen 4  -> 22 chars total

// 1) One frame split across two events -> reassembled into one whole frame.
assert.deepStrictEqual(
  collectFrames([A.slice(0, 20), A.slice(20)]), [A],
  'split frame should be reassembled');

// 2) Two frames coalesced in one event -> split back into two whole frames.
assert.deepStrictEqual(
  collectFrames([A + B]), [A, B],
  'coalesced frames should be separated');

// 3) Header itself split (<18 chars) then the rest -> waits, then one frame.
assert.deepStrictEqual(
  collectFrames([A.slice(0, 10), A.slice(10)]), [A],
  'partial header should wait for more bytes');

// 4) Non-3E mode is untouched: raw chunk passes straight through to onResponse.
(function () {
  const conn = new MC();
  const sock = new EventEmitter();
  sock.remoteAddress = '127.0.0.1';
  sock.remotePort = 0;
  conn.isoclient = sock;
  conn.isAscii = false;
  conn.frame = '1E';
  let got = null;
  conn.onResponse = (d) => { got = Buffer.from(d).toString('ascii'); };
  const log = console.log;
  console.log = () => {};
  try {
    conn.onTCPConnect();
    sock.emit('data', Buffer.from('rawbytes', 'ascii'));
  } finally {
    console.log = log;
  }
  assert.strictEqual(got, 'rawbytes', '1E path should pass data through unchanged');
})();

console.log('mcprotocol reassembly: all 4 checks passed');
