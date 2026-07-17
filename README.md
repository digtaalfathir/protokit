# Protokit

> A kit of device communication protocols.

Industrial device comms (Modbus, MC, Open Protocol) — not related to protobuf tooling.

Protokit is a monorepo of small, focused Node.js libraries for talking to
industrial devices (PLCs, controllers, assembly tools). Each protocol is
published as its own npm package under the [`@digta`](https://www.npmjs.com/org/digta)
scope, so you install only the protocol you need.

## Packages

| Package | Protocol | Status | npm |
|---|---|---|---|
| [@digta/fins](packages/fins) | Omron FINS TCP | ✅ Working (app/service) | `npm i @digta/fins` |
| [@digta/modbus](packages/modbus) | Modbus TCP/RTU | 🚧 Placeholder (demo in `examples/`) | `npm i @digta/modbus` |
| [@digta/mcprotocol](packages/mcprotocol) | Mitsubishi MELSEC MC (SLMP) | 🚧 Placeholder | `npm i @digta/mcprotocol` |
| [@digta/openprotocol](packages/openprotocol) | Atlas Copco Open Protocol | 🚧 Placeholder | `npm i @digta/openprotocol` |

**Status legend:** ✅ usable today · 🚧 scaffolded, implementation in progress.

> **Note:** `@digta/openprotocol` is the Atlas Copco assembly-tool protocol,
> **not** Google Protocol Buffers.

## Install

Install any package on its own:

```bash
npm i @digta/fins
npm i @digta/modbus
npm i @digta/mcprotocol
npm i @digta/openprotocol
```

## Development

This repo uses **native npm workspaces** (no Lerna/Nx/Turborepo).

```bash
git clone https://github.com/digtaalfathir/protokit.git
cd protokit
npm install            # installs all workspaces + links them together
```

Common workspace commands:

```bash
npm install                          # install deps for every package
npm run start -w @digta/fins         # run a script in one package
npm test --workspaces --if-present   # run tests across all packages
```

## How to add a protocol package

1. Create the folder: `packages/<name>/`.
2. Add `packages/<name>/package.json`:
   - `"name": "@digta/<name>"`, `"version": "0.1.0"`
   - `description` + `keywords` (protocol, industrial, plc, iot, tcp, serial, …)
   - `"license": "MIT"`, `"author": "Rifky Andigta Al-Fathir"`
   - `main` / `exports` and a `"files"` field (publish only what's needed)
   - `repository`: `{ "type": "git", "url": "git+https://github.com/digtaalfathir/protokit.git", "directory": "packages/<name>" }`
3. Add `packages/<name>/index.js` (the entrypoint) and `packages/<name>/README.md`
   (what the protocol is, install, a short usage example).
4. Run `npm install` at the repo root to wire the new workspace in.
5. Add a row to the [Packages](#packages) table above.

## Publishing

Packages are **not** published automatically. To publish (or update) a package:

**Prerequisites (one-time):**

- Create the npm organization **`digta`** (npmjs.com → *Add Organization*).
- Log in locally: `npm login`.

**Publish a package.** Scoped packages default to **private**, so you must pass
`--access public`:

```bash
npm publish -w @digta/<name> --access public
```

For example:

```bash
npm publish -w @digta/fins --access public
```

## License

[MIT](LICENSE) © Rifky Andigta Al-Fathir
