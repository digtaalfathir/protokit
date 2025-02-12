# Omron FINS TCP for Data Acquisition

Omron Fins TCP is a protocol for communication between Omron PLCs and other devices. It is a TCP/IP based protocol. The fins plugin is used for Omron PLCs with network port, such as CP2E.


## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Service Management](#Service-Management-(Linux))
- [Troubleshooting](#troubleshooting)

## Prerequisites

Ensure you have the following installed:

- Node.js (v14 or higher)
- npm (Node Package Manager)
- Omron FINS Node.js library
- CX Programmer (tested on v9.7)

## Installation

1. Clone the repository:

    ```bash
    git clone https://gitlab.com/source-code-documentation-hw-stechoq/playground/plc2pc-protocol.git
    git checkout OmronFinsTCP
    ```

2. Install the necessary Node.js packages:

    ```bash
    npm install
    ```

3. Run the installation script as root to set up the systemd service:

    ```bash
    sudo ./install.sh
    ```

## Configuration

The application configuration is handled in the `app.config.sample.js` file. Copy this file and rename it to `app.config.js`. Modify the values according to your setup.

```javascript
module.exports = {
  plc: {
    ip: '172.19.88.88',  // IP address of the PLC
    intervalRead: 200   // Interval for reading the PLC (in milliseconds)
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
type in terminal, make sure you in right dir
```
node index.js
```

## Service Management (Linux)
 ```
sudo systemctl start plcdcs.service  --> to start the service
sudo systemctl enable plcdcs.service --> to enable service, so it can auto-start
sudo systemctl status plcdcs.service --> to check the status of service
```

## Troubleshooting
```
sudo journalctl -u plcdcs.service
```