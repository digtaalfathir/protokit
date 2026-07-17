# Panduan Publish ke npm (@digta)

Panduan lengkap dari nol sampai paket bisa diakses online. Ditulis untuk publish
pertama kali. Ikuti berurutan.

Yang sudah siap: semua paket (`@digta/fins`, `@digta/mcprotocol`,
`@digta/open-protocol`) sudah punya `name`, `version`, `license`, `files`, dan
`repository`. Tinggal buat akun + publish. (`@digta/modbus` masih placeholder —
jangan dipublish dulu.)

---

## 1. Buat akun npm

1. Daftar di <https://www.npmjs.com/signup>.
2. **Verifikasi email** lewat link yang dikirim npm. Ini wajib — kalau email
   belum terverifikasi, `npm publish` akan ditolak.

## 2. Aktifkan 2FA (Two-Factor Authentication)

npm meminta 2FA untuk publish.

1. npmjs.com → foto profil (kanan atas) → **Account** → **Two-Factor
   Authentication** → aktifkan untuk **"Authorization and publishing"**.
2. Scan QR pakai app authenticator (Google Authenticator / Authy).
3. Simpan recovery codes.

Nanti tiap publish akan diminta **OTP** (6 digit dari app).

## 3. Buat organization "digta"

Nama paket `@digta/...` berarti scope **`digta`** harus kamu miliki.

1. npmjs.com → foto profil → **Add Organization**.
2. Nama organization: **`digta`**.
3. Pilih plan **Free** (cukup untuk paket publik).

> Kalau nama `digta` ternyata sudah dipakai orang lain, kamu harus ganti scope
> (misal `@digtaalfathir`) di **semua** `packages/*/package.json` pada field
> `name`, lalu `npm install` lagi.

## 4. Login di terminal

```bash
npm login
```

Masukkan username, password, email, lalu OTP. Cek berhasil:

```bash
npm whoami        # harus muncul username kamu
```

## 5. Cek isi paket dulu (dry-run, tidak mengirim apa pun)

Dari root repo:

```bash
npm pack --dry-run -w @digta/fins
```

Pastikan file yang keluar hanya yang perlu (sesuai field `files`). Ulangi untuk
`@digta/mcprotocol` dan `@digta/open-protocol`.

## 6. Publish

Paket scoped **default-nya private**. Untuk publik **wajib** `--access public`:

```bash
npm publish -w @digta/fins --access public
npm publish -w @digta/mcprotocol --access public
npm publish -w @digta/open-protocol --access public
```

- Tiap publish akan minta OTP. Kalau tidak muncul prompt, tambahkan manual:
  `npm publish -w @digta/fins --access public --otp=123456`.
- **Tip:** biar tak perlu mengetik `--access public` tiap kali, tambahkan ke
  `package.json` masing-masing paket:
  ```json
  "publishConfig": { "access": "public" }
  ```

## 7. Verifikasi sudah online

```bash
npm view @digta/fins           # tampilkan metadata versi terbaru
```

Atau buka di browser: <https://www.npmjs.com/package/@digta/fins>

Tes install di folder kosong:

```bash
mkdir /tmp/coba && cd /tmp/coba && npm init -y && npm i @digta/fins
```

## 8. Update versi (untuk rilis berikutnya)

Versi yang sama **tidak bisa** dipublish dua kali. Untuk update:

```bash
cd packages/fins
npm version patch     # 1.0.0 -> 1.0.1 (patch/minor/major)
cd ../..
npm publish -w @digta/fins --access public
```

---

## Error yang sering muncul

| Pesan | Artinya | Solusi |
|-------|---------|--------|
| `402 Payment Required` / *private* | lupa akses publik | tambah `--access public` |
| `403 Forbidden` | belum member org / scope bukan milikmu / email belum verifikasi | cek org `digta`, verifikasi email |
| `EOTP` / `one-time password` | butuh 2FA | tambah `--otp=123456` |
| `409` / *cannot publish over previously published version* | versi sudah ada | `npm version patch` dulu |
| `ENEEDAUTH` | belum login | `npm login` |
| `E404` saat install | nama/scope salah atau belum publish | cek nama paket persis |
