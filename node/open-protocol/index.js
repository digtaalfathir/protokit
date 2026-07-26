'use strict';

// Public entry: the low-level codec (buildFrame, buildMID*, parseMID*, getReplyMID)
// plus the high-level OpenProtocolClient. require("@digta/open-protocol") gives both.
const codec = require('./openProtocol');
const OpenProtocolClient = require('./client');

module.exports = Object.assign({}, codec, { OpenProtocolClient });
