#!/usr/bin/env node

const fins = require("omron-fins");
// const Netcat = require('node-netcat')
const net = require("net");
const date = require("date-and-time");

const config = require("./config/app.config");

const options = { timeout: 5000, SA1: 0, DA1: 10, protocol: "tcp" }; //protocol can be "udp" or "tcp" only
const IP = config.plc.ip;
const PORT = 9600;
const client = fins.FinsClient(PORT, IP, options);
let readyToSend = Array(8).fill(true);
let timerSendLast = Array(8).fill(0);

const machineId = config.machineId;

let ncconnect = false;
let sensorValue = [];
let initialCondition = true;
let timerSend = [];
// let timerSendLast=[];
// Connecting / disconnecting...
const optionsnc = {
  // define a connection timeout
  timeout: 1000,
  // buffer(default, to receive the original Buffer objects), ascii, hex,utf8, base64
  read_encoding: "buffer",
};
const HOSTIP = config.dcs.ip;
const HOSTPORT = config.dcs.port;
let msg = { response: { values: new Array(8) } };

const intervaltimerSend = {
  interval: config.intervalBouncing,
};

(async () => {
  console.log("connect to dcs");
  if (initialCondition == true) {
    console.log(config.plc.ip);
    // readyToSend[1]=true;
    initialCondition = false;
  }
})();

function sendData(__index) {
  if (new Date() - timerSendLast[__index] > intervaltimerSend.interval && readyToSend[__index] == true) {
    readyToSend[__index] = false;
    const nc = new net.Socket();
    nc.connect(HOSTPORT, HOSTIP, function () {
      console.log("🚀 ~ file: index.js:52 ~ sendData ~ __index:", config.msg[__index]);
      config.msg[__index].id = config.machineId;
      const data_send = `${JSON.stringify(config.msg[__index])}`;
      const id = setTimeout(() => {
        nc.write(data_send);
        clearTimeout(id);
        nc.destroy();
      }, 1000);
      console.log("🚀 ~ file: index.js:32 ~ connect:", data_send);
      timerSendLast[__index] = new Date();
    });

    nc.on("error", function (err) {
      // console.log(err.code);
      console.log("🚀 ~ file: index.js:52 ~ nc.on ~ err:", err);
      throw new Error(err.code);
    });

    nc.on("close", function () {
      console.log("🚀 ~ file: index.js:40 ~ nc.on ~ close");
    });
  }
}

// client init connection

client.connect();

// client.connect()?console.log('not connect'):console.log('connect');

client.on("error", function (error, msg) {
  console.log("Error: ", error, msg);
});

setInterval(() => {
  client.read(`D10000`, 10, function (err, msg) {
    if (err) throw new Error("plc not connected");
    // console.log("🚀 ~ file: index.js:29 ~ client..readMultiple ~ msg:", msg)

    sensorValue = msg.response.values;
    for (const index in sensorValue) {
      // console.log(`${sensorValue[index]} is at position ${index}`)
      if (sensorValue[index] == 1) {
        sendData(index);
      } else {
        readyToSend[index] = true;
      }
    }

    // console.log("🚀 ~ file: index.js:32 ~ client.read ~ sensorValue:", sensorValue)
  });
}, config.plc.intervalRead);
