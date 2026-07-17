# @digta/mcprotocol

Mitsubishi **MELSEC Communication (MC) protocol** client for Ethernet PLCs,
written entirely in JavaScript.

This is an `@digta`-scoped **fork of [`mcprotocol`](https://www.npmjs.com/package/mcprotocol) 0.1.2**
(by Dana Moffit) with a single behavioural fix: **TCP frame reassembly for ASCII
3E responses**. It is a drop-in replacement — no API changes. See
[CHANGELOG.md](./CHANGELOG.md) for the details.

Part of [Protokit](../../README.md).

> This software is not affiliated with Mitsubishi. FX3U and MELSEC are
> trademarks of Mitsubishi.

> ⚠️ **BETA CODE.** Wrong values can be written to wrong locations. Test
> everything thoroughly before using against a live PLC.

## Install

```bash
npm i @digta/mcprotocol
```

## What's different from upstream

`mcprotocol@0.1.2` treated every socket `data` event as exactly one complete PLC
response. TCP is a byte stream, so an ASCII 3E response can be split across
several events — or several responses can arrive in one — which produced
`Invalid Response Length` / `DATA LESS THAN 11 BYTES` errors and connection
resets when polling many registers.

This fork buffers incoming bytes per connection and slices out whole frames
using the response data-length field (`18 + dataLen` chars) before processing.
It is active **only** for `ascii: true` + `frame: '3E'`; every other mode uses
the original code path. Full write-up in [CHANGELOG.md](./CHANGELOG.md).

## Usage

```js
var mc = require('@digta/mcprotocol');
var conn = new mc;

var variables = {
  TEST1: 'D0,5',        // 5 words starting at D0
  TEST4: 'R2000,2',     // 2 words at R2000
  TEST6: 'D6000.1,20',  // 20 bits starting at D6000.1
};

conn.initiateConnection({ port: 1281, host: '192.168.0.2', ascii: false }, connected);

function connected(err) {
  if (typeof err !== 'undefined') { console.log(err); process.exit(); }
  conn.setTranslationCB(function (tag) { return variables[tag]; });
  conn.addItems(['TEST1', 'TEST4']);
  conn.addItems('TEST6');
  conn.readAllItems(valuesReady);
}

function valuesReady(anythingBad, values) {
  if (anythingBad) { console.log('SOMETHING WENT WRONG READING VALUES'); }
  console.log(values);
  process.exit();
}
```

> Note: to use the ASCII 3E fix, configure the connection with
> `{ ascii: true, frame: '3E' }`.

## API

- `initiateConnection(params, callback)` — connect. `params`: `port`, `host`,
  `ascii` (default `false`), `frame` (`'1E'` default, `'3E'` supported),
  `octalInputOutput` (default `true`). `callback(err)`.
- `dropConnection()` — terminate the TCP connection.
- `setTranslationCB(translator)` — map tag names to addresses. `translator(tag)`
  returns an address string, e.g. `M100,20`, `D2000,5`, `DFLOAT1000`,
  `RSTR30,10`, `D1000.2,5`.
- `addItems(items)` / `removeItems(items)` — manage the read polling list
  (string or array of strings).
- `writeItems(items, values, callback)` — write values to the PLC.
- `readAllItems(callback)` — read the polling list; `callback(anythingBad, values)`.

For the full address syntax and configurator setup notes, see the
[upstream mcprotocol documentation](https://www.npmjs.com/package/mcprotocol).

## Tested against

Direct connection to FX3U-ENET / FX3U-ENET-ADP. Q-series E71 should work in
theory (same frames). Serial and UDP are not supported.

## Changelog

`1.0.0` — fork of `mcprotocol@0.1.2` with the ASCII 3E TCP frame reassembly fix.
Full details in [CHANGELOG.md](./CHANGELOG.md).

## License

[MIT](LICENSE) — © 2015 Dana Moffit (original), © 2026 Rifky Andigta Al-Fathir
(fork modifications).
