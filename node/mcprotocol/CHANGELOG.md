# Changelog

## 1.0.0 (fork dari mcprotocol@0.1.2)

Fork internal. Satu perubahan terhadap 0.1.2 (di `mcprotocol.js`, handler
`isoclient.on('data')` dalam `onTCPConnect`):

**Fix: reassembly frame TCP untuk respons ASCII 3E.**

Library asli menganggap tiap event `data` dari socket = tepat satu respons PLC
utuh. TCP itu byte-stream, jadi satu respons bisa terpecah ke beberapa event
atau beberapa respons menempel dalam satu event. Akibatnya, saat membaca banyak
register, muncul error `Invalid Response Length` / `DATA LESS THAN 11 BYTES` dan
koneksi terus reset.

Fix menampung byte masuk di buffer per-koneksi (`self._recvBuffer`) dan memotong
per frame utuh sebelum diteruskan ke `onResponse`. Panjang frame dihitung dari
field data-length respons ASCII 3E: 4 char hex big-endian pada char 14–18 =
jumlah karakter setelahnya, jadi total frame = `18 + dataLen` char.

Hanya aktif untuk `ascii: true` + `frame: '3E'`; mode lain memakai jalur lama.
Tidak ada perubahan API — drop-in replacement untuk `mcprotocol@0.1.2`.

---

## 0.1.2 (upstream asli)

- Underscore vulnerability fix
- Previous version changes not recorded.
