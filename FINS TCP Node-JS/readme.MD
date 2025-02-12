# omron fins

this program used to connect with PLC Omron CP2E using fins protocol 

## Installation Step

1. copy app.config.sample.js -> app.config.js
2. change parameter of config file 
3. IP PLC must be 1 class with PC
>sudo nmtui
4. run ```install.sh``` file
>sudo ./install.sh

### config file
```
  plc :{
    ip:'172.19.88.88', // ip PLC 
    intervalRead : 200 // interval plc reading 
  },
  machineId:27, //machine Id PLC
  msg :[

    {proxy:1},//message pin 00.00
    {proxy:1},//message pin 00.01
    {proxy:1},//message pin 00.02
    {proxy:1},//message pin 00.03
    {proxy:1},//message pin 00.04
    {proxy:1},//message pin 00.05
    {proxy:1},//message pin 00.06
    {proxy:1} //message pin 00.07
    ],
  intervalBouncing : 5000, //delay between shoot
  dcs :{
    ip:'127.0.0.1', //localhost socket IP 
    port : 3000 //localhost socket port
  }
  ```