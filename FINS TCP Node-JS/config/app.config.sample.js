module.exports = {
  plc: {
    ip: "192.168.250.1",
    intervalRead: 200,
  },
  machineId: 27,
  msg: [
    { proxy: 1 }, //pin 00.00
    { proxy: 1 }, //pin 00.01
    { proxy: 1 }, //pin 00.02
    // {proxy:1},//pin 00.03
    // {proxy:1},//pin 00.04
    // {proxy:1},//pin 00.05
    // {proxy:1},//pin 00.06
    // {proxy:1}//pin 00.07
  ],
  intervalBouncing: 5000, //delay between shoot
  dcs: {
    ip: "127.0.0.1",
    port: 3000,
  },
};
