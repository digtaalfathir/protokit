#!/usr/bin/env node

const fins = require("omron-fins");
const net = require("net");

const config = require("./config/app.config");

const options = { timeout: 5000, SA1: 0, DA1: 10, protocol: "tcp" }; // protocol: "udp" or "tcp"
const PLC_IP = config.plc.ip;
const FINS_PORT = 9600;
const client = fins.FinsClient(FINS_PORT, PLC_IP, options);

const HOSTIP = config.dcs.ip;
const HOSTPORT = config.dcs.port;
const debounceMs = config.intervalBouncing;

// One slot per DCS message channel.
const channelCount = config.msg.length;
const readyToSend = Array(channelCount).fill(true);
const timerSendLast = Array(channelCount).fill(0);

console.log(`FINS acquisition: PLC ${PLC_IP}:${FINS_PORT} -> DCS ${HOSTIP}:${HOSTPORT}, ${channelCount} channels`);

function sendData(index) {
  if (readyToSend[index] !== true) return;
  if (new Date() - timerSendLast[index] <= debounceMs) return;

  // Latch the channel and stamp the attempt up front so a failed send retries
  // only after the debounce window (not on every read tick).
  readyToSend[index] = false;
  timerSendLast[index] = new Date();

  const nc = new net.Socket();

  nc.connect(HOSTPORT, HOSTIP, function () {
    config.msg[index].id = config.machineId;
    const data_send = JSON.stringify(config.msg[index]);
    // brief settle, then send and close
    setTimeout(() => {
      nc.write(data_send);
      nc.destroy();
    }, 1000);
    console.log(`channel ${index} -> DCS: ${data_send}`);
  });

  nc.on("error", function (err) {
    // DCS unreachable: log and re-arm this channel so the event retries after
    // the debounce window, instead of crashing the whole service.
    console.error(`DCS send error (channel ${index}): ${err.message}`);
    readyToSend[index] = true;
    nc.destroy();
  });
}

client.connect();

client.on("error", function (error) {
  console.error("FINS client error:", error);
});

setInterval(() => {
  client.read("D10000", channelCount, function (err, msg) {
    if (err) {
      // Transient PLC read failure: log and try again next tick, don't crash.
      console.error("PLC read failed:", err.message || err);
      return;
    }
    const sensorValue = msg.response.values;
    for (let index = 0; index < channelCount; index++) {
      if (sensorValue[index] == 1) {
        sendData(index);
      } else {
        readyToSend[index] = true;
      }
    }
  });
}, config.plc.intervalRead);
