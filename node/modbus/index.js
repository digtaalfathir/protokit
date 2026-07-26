'use strict';

const ModbusTcpClient = require('./tcp-client');
const { ModbusError } = require('./errors');

module.exports = { ModbusTcpClient, ModbusError };
