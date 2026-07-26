# Protokit

> A kit of device communication protocols.

Industrial device comms (Modbus, MC, Open Protocol) — **multi-language**, not related to protobuf tooling.

Protokit is a monorepo of small, focused libraries and services for talking to
industrial devices (PLCs, controllers, assembly tools), organized **by language**:

```
protokit/
  node/   npm workspaces — @digta/* packages (published to npm)
  go/     Go module — libraries + runnable commands (compiled binaries)
```

Each language uses its own native tooling — npm for `node/`, Go modules for
`go/` — so they don't step on each other.

## Node.js packages (`node/`)

Published under the [`@digta`](https://www.npmjs.com/org/digta) scope; install
only the protocol you need.

| Package | Protocol | Status | npm |
|---|---|---|---|
| [@digta/fins](node/fins) | Omron FINS TCP | ✅ Working (app/service) | `npm i @digta/fins` |
| [@digta/modbus](node/modbus) | Modbus TCP | ✅ Working (zero-dep TCP master) | `npm i @digta/modbus` |
| [@digta/mcprotocol](node/mcprotocol) | Mitsubishi MELSEC MC (1E/3E) | ✅ Working (fork of mcprotocol 0.1.2 + ASCII 3E framing fix) | `npm i @digta/mcprotocol` |
| [@digta/open-protocol](node/open-protocol) | Atlas Copco Open Protocol | ✅ Working (zero-dep MID codec + TCP client) | `npm i @digta/open-protocol` |

**Status legend:** ✅ usable today · 🚧 scaffolded, implementation in progress.

> **Note:** `@digta/open-protocol` is the Atlas Copco assembly-tool protocol,
> **not** Google Protocol Buffers.

## Go (`go/`)

🚧 **Skeleton.** The Go module is set up; protocol implementations are being
added. Go compiles to a single static binary (no runtime needed on the target).
See **[go/README.md](go/README.md)** for the layout and how to build, run, and
autostart (systemd / pm2).

## Install (Node.js)

```bash
npm i @digta/fins
npm i @digta/modbus
npm i @digta/mcprotocol
npm i @digta/open-protocol
```

## Development

### Node.js (`node/`)

Native **npm workspaces** (no Lerna/Nx/Turborepo).

```bash
git clone https://github.com/digtaalfathir/protokit.git
cd protokit
npm install                          # install + link all node/ workspaces
npm run start -w @digta/fins         # run a script in one package
npm test --workspaces --if-present   # run tests across all packages
```

### Go (`go/`)

```bash
cd go
go run ./cmd/<tool>                  # run during development
go build -o <tool> ./cmd/<tool>      # build a binary
GOOS=linux GOARCH=amd64 go build ...  # cross-compile for a Linux device
```

## How to add a protocol

### A Node.js package (`node/`)

1. Create `node/<name>/`.
2. Add `node/<name>/package.json`:
   - `"name": "@digta/<name>"`, `"version": "0.1.0"`
   - `description` + `keywords` (protocol, industrial, plc, iot, tcp, serial, …)
   - `"license": "MIT"`, `"author": "Rifky Andigta Al-Fathir"`
   - `main` / `exports` and a `"files"` field (publish only what's needed)
   - `repository`: `{ "type": "git", "url": "git+https://github.com/digtaalfathir/protokit.git", "directory": "node/<name>" }`
3. Add `node/<name>/index.js` (entrypoint) and `node/<name>/README.md`.
4. Run `npm install` at the repo root to wire the new workspace in.
5. Add a row to the [Node.js packages](#nodejs-packages-node) table above.

### A Go package (`go/`)

Add a library under `go/<name>/` and/or a runnable command under
`go/cmd/<tool>/`. See [go/README.md](go/README.md).

## Publishing (npm)

📖 **First time? Full step-by-step walkthrough: [PUBLISHING.md](PUBLISHING.md).**

Node packages are **not** published automatically. Prerequisites: create the npm
org **`digta`** and `npm login`. Scoped packages default to **private**, so pass
`--access public`:

```bash
npm publish -w @digta/<name> --access public
```

## License

[MIT](LICENSE) © Rifky Andigta Al-Fathir
