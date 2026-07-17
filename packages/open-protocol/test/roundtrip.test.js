'use strict';

// Round-trip checks for the Open Protocol builder/parser. No PLC/tool needed.
const assert = require('assert');
const op = require('../openProtocol.js');

// All 9 documented exports are present and callable.
const EXPORTS = [
  'buildFrame', 'buildMID0001', 'buildMID0050', 'buildMID0060', 'buildMID0062',
  'buildMID9999', 'parseMID', 'parseMID0061', 'getReplyMID',
];
EXPORTS.forEach((name) => assert.strictEqual(typeof op[name], 'function', `missing export: ${name}`));

// buildFrame -> parseMID round-trip.
const frame = op.buildFrame('0050', '001', 'BODY');
const p = op.parseMID(frame);
assert.strictEqual(p.mid, '0050', 'mid round-trips');
assert.strictEqual(p.revision, '001', 'revision round-trips');
assert.strictEqual(p.body, 'BODY', 'body round-trips');
// length = 20 header + 4 body + 1 NUL = 25 -> "0025"
assert.strictEqual(p.length, '0025', 'length header is computed correctly');

// A canned builder parses back to its own MID.
assert.strictEqual(op.parseMID(op.buildMID0001()).mid, '0001', 'MID0001 parses');
assert.strictEqual(op.parseMID(op.buildMID9999()).mid, '9999', 'MID9999 parses');

// getReplyMID pulls the acknowledged MID out of the body (chars 20-24).
const reply = op.buildFrame('0005', '001', '0001'); // "command accepted" carrying 0001
assert.strictEqual(op.getReplyMID(op.parseMID(reply).raw), '0001', 'getReplyMID extracts replied MID');

// parseMID0061 returns the documented shape without throwing on empty fields.
const r = op.parseMID0061(' '.repeat(200));
assert.ok(r && typeof r === 'object', 'parseMID0061 returns an object');
['vin', 'torqueValue', 'angleValue', 'tighteningStatus'].forEach(
  (k) => assert.ok(k in r, `parseMID0061 result has key: ${k}`));

console.log('open-protocol round-trip: all checks passed');
