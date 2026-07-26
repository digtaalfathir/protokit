'use strict';

// Standard Modbus exception codes (from the response when the high bit of the
// function code is set).
const EXCEPTION_MESSAGES = {
  1: 'Illegal function',
  2: 'Illegal data address',
  3: 'Illegal data value',
  4: 'Server device failure',
  5: 'Acknowledge',
  6: 'Server device busy',
  8: 'Memory parity error',
  10: 'Gateway path unavailable',
  11: 'Gateway target device failed to respond',
};

/**
 * Error thrown for Modbus-level failures. `code` is the numeric exception code
 * for protocol exceptions, or a string tag for transport issues
 * ('ETIMEDOUT', 'ENOTCONN', 'ECLOSED', 'EMISMATCH').
 */
class ModbusError extends Error {
  constructor(message, code) {
    super(message);
    this.name = 'ModbusError';
    this.code = code;
  }
}

module.exports = { ModbusError, EXCEPTION_MESSAGES };
