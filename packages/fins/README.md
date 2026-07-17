# @digta/fins — Omron FINS TCP for Data Acquisition

Omron FINS TCP is a protocol for communication between Omron PLCs and other devices. It is a TCP/IP based protocol, used for Omron PLCs with a network port such as the CP2E.

This package reads PLC inputs over FINS TCP and forwards them to a DCS over a plain TCP socket. It ships as a runnable app/service, not a library API.

Part of [Protokit](../../README.md).

## Install

```bash
npm i @digta/fins
```

Or run it from a clone of the monorepo (see [dev setup](../../README.md#development)).

## Table of Contents

- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
- [Usage](#usage)
- [Service Management (Linux)](#service-management-linux)
- [Troubleshooting](#troubleshooting)

## Prerequisites

Ensure you have the following installed:

- Node.js (v14 or higher)
- npm (Node Package Manager)
- CX Programmer (tested on v9.7) for the PLC side
- The PLC IP must be on the same subnet as the PC (`sudo nmtui`)

## Configuration

Copy the sample config and edit it for your setup:

```bash
cp config/app.config.sample.js config/app.config.js
```

```javascript
module.exports = {
  plc: {
    ip: '172.19.88.88',  // IP address of the PLC
    intervalRead: 200    // Interval for reading the PLC (in milliseconds)
  },
  machineId: 27,  // Machine ID (DCS)
  msg: [
    { proxy: 1 }, // pin 00.00
    { proxy: 1 }, // pin 00.01
    { proxy: 1 }, // pin 00.02
    { proxy: 1 }, // pin 00.03
    { proxy: 1 }, // pin 00.04
    { proxy: 1 }, // pin 00.05
    { proxy: 1 }, // pin 00.06
    { proxy: 1 }  // pin 00.07
  ],
  intervalBouncing: 5000,  // Delay between data transmissions
  dcs: {
    ip: '127.0.0.1',  // IP address of the DCS (localhost)
    port: 3000        // Port of the DCS
  }
};
```

## Usage

Make sure you are in the package directory, then:

```bash
node index.js
```

## Service Management (Linux)

Run the installer as root to set up the systemd service:

```bash
sudo ./install.sh
```

```bash
sudo systemctl start plcdcs.service   # start the service
sudo systemctl enable plcdcs.service  # enable auto-start on boot
sudo systemctl status plcdcs.service  # check status
```

## Troubleshooting

```bash
sudo journalctl -u plcdcs.service
```
