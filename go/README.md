# Protokit — Go

The Go side of [Protokit](../README.md): device-communication libraries and
runnable services, compiled to **single static binaries** (no runtime needed on
the target — you copy one file and run it).

## Packages

| Package | Protocol | Status |
|---|---|---|
| [mcprotocol](mcprotocol) | Mitsubishi MELSEC MC (3E binary, word devices) | ✅ read/write words |
| [modbus](modbus) | Modbus TCP (master) | ✅ read/write coils + registers |
| [open-protocol](open-protocol) | Atlas Copco Open Protocol | ✅ codec + client (reconnect/heartbeat) |
| [fins](fins) | Omron FINS/TCP | ✅ read/write memory areas |

All are Go ports of the matching `node/` packages, zero external dependencies.
Runnable commands live in `cmd/` — e.g. `go run ./cmd/mcread`.

## Layout

```
go/
  go.mod                 module: github.com/digtaalfathir/protokit/go
  <protocol>/            library package  (e.g. modbus/, fins/)
  cmd/<tool>/main.go     runnable command → builds to one binary
```

- **Libraries** live in `go/<protocol>/` and are imported as
  `github.com/digtaalfathir/protokit/go/<protocol>`.
- **Runnable services** live in `go/cmd/<tool>/`. Each `cmd/<tool>` builds to a
  standalone binary you deploy and run.

## Build & run

```bash
cd go

go run ./cmd/<tool>                       # run during development (compile + run)
go build -o <tool> ./cmd/<tool>           # build a binary for this machine
GOOS=linux GOARCH=amd64 go build -o <tool> ./cmd/<tool>   # cross-compile for a Linux device
```

The output is one file (`<tool>`). Deploy = copy that file to the server. No
`node_modules`, no interpreter install.

## Autostart & logs

A Go binary doesn't need a process manager to *run* (it's not a script). To keep
it alive across reboots/crashes and to read its logs, use **systemd** (native)
or **pm2** (if you already know it).

### systemd (recommended) — same pattern as the FINS service

`/etc/systemd/system/protokit-<tool>.service`:

```ini
[Unit]
Description=Protokit <tool>
After=network.target

[Service]
ExecStart=/opt/protokit/<tool>
Restart=always
RestartSec=5
Environment=PLC_HOST=192.168.1.10
WorkingDirectory=/opt/protokit

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now protokit-<tool>   # autostart on boot + start now
journalctl -u protokit-<tool> -f              # follow logs
sudo systemctl restart protokit-<tool>        # restart
systemctl status protokit-<tool>              # status
```

### pm2 (if you prefer what you already know)

pm2 can supervise any binary, not just Node:

```bash
go build -o <tool> ./cmd/<tool>
pm2 start ./<tool> --name <tool> --interpreter none
pm2 logs <tool>
pm2 startup && pm2 save
```

Trade-off: pm2 needs Node installed on the target just to babysit a binary that
otherwise wouldn't. systemd has no such dependency.

## Coming from Node/pm2 — quick map

| pm2 | systemd |
|-----|---------|
| `pm2 start app.js` | `systemctl start <svc>` (ExecStart → binary) |
| `pm2 startup` + `pm2 save` | `systemctl enable <svc>` |
| `pm2 logs` | `journalctl -u <svc> -f` |
| `pm2 restart` | `systemctl restart <svc>` |
| `pm2 list` / `status` | `systemctl status <svc>` |
| restart-on-crash | `Restart=always` |

## Logging

No extra library needed. Use the standard library: `log`, or `log/slog`
(structured, Go 1.21+). Write to stdout/stderr — systemd's journald (or pm2)
captures it automatically:

```go
slog.Info("connected", "host", host, "port", port)
```
