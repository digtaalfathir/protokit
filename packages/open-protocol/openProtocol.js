/**
 * Open Protocol Helper
 *
 * Centralizes frame construction, parsing, and MID definitions
 * for Atlas Copco Open Protocol communication.
 *
 * Frame format (20-byte header):
 *   Bytes 0-3   : Length (4 digits, includes header + body + NUL)
 *   Bytes 4-7   : MID (4 digits)
 *   Bytes 8-10  : Revision (3 digits)
 *   Bytes 11-12 : NoAck flag (2 digits, unused here)
 *   Bytes 13-14 : Station ID (2 digits, unused here)
 *   Bytes 15-16 : Spindle ID (2 digits, unused here)
 *   Bytes 17-19 : Spare (3 digits)
 *   Byte  20    : NUL terminator
 */

const HEADER_LENGTH = 20;
const NUL = "\x00";

// --- Frame Builder ---

/**
 * Build a raw Open Protocol frame string.
 * @param {string} mid - 4-digit MID (e.g. "0001")
 * @param {string} [revision="001"] - 3-digit revision
 * @param {string} [body=""] - Optional body after header
 * @returns {Buffer}
 */
function buildFrame(mid, revision = "001", body = "") {
  const totalLength = HEADER_LENGTH + body.length + 1; // +1 for NUL
  const lengthStr = String(totalLength).padStart(4, "0");
  const revStr = String(revision).padStart(3, "0");
  const header = `${lengthStr}${mid}${revStr}${" ".repeat(9)}`;
  return Buffer.from(header + body + NUL, "ascii");
}

// --- MID Builders ---

/** MID 0001 - Communication Start */
function buildMID0001() {
  return Buffer.from("00200001001         \x00", "ascii");
}

/** MID 0050 - Vehicle ID Number Download Request */
function buildMID0050(vin) {
  const vinStr = vin.padEnd(25, " ").substring(0, 25);
  const frameStr = `00450050            ${vinStr}\x00`;
  return Buffer.from(frameStr, "ascii");
}

/** MID 0060 - Last Tightening Result Data Subscribe */
function buildMID0060() {
  // Revision 001, extra spacing for subscribe frame
  return Buffer.from(`002000600011        ${NUL}`, "ascii");
}

/** MID 0062 - Last Tightening Result Data Acknowledge */
function buildMID0062() {
  return Buffer.from("00200062            \x00", "ascii");
}

/** MID 9999 - Alive / Keep-Alive (Heartbeat) */
function buildMID9999() {
  return Buffer.from("00209999001         \x00", "ascii");
}

// --- Parser ---

/**
 * Parse an Open Protocol frame from raw ASCII data.
 * @param {string|Buffer} data
 * @returns {{ raw: string, length: string, mid: string, revision: string, body: string }}
 */
function parseMID(data) {
  const msg = (Buffer.isBuffer(data) ? data.toString("ascii") : data).replace(
    /\x00/g,
    ""
  );

  return {
    raw: msg,
    length: msg.substring(0, 4),
    mid: msg.substring(4, 8),
    revision: msg.substring(8, 11),
    body: msg.substring(HEADER_LENGTH),
  };
}

/**
 * Extract the accepted/rejected MID from a MID 0005 (Command Accepted)
 * or MID 0004 (Command Error) reply.
 * @param {string} raw - Full frame string (NUL-stripped)
 * @returns {string} 4-digit MID that was accepted/rejected
 */
function getReplyMID(raw) {
  return raw.substring(20, 24);
}

/**
 * Parse MID 0061 - Last Tightening Result Data
 * @param {string} data - Raw ASCII frame
 * @returns {object} Parsed tightening result fields
 */
function parseMID0061(data) {
  const clean = (typeof data === "string" ? data : data.toString("ascii")).replace(
    /\x00/g,
    ""
  );

  const safeParse = (val, divisor = 1) => {
    const num = parseInt(val);
    return isNaN(num) ? null : num / divisor;
  };

  return {
    vin: clean.substring(59, 84).trim(),

    pSet: safeParse(clean.substring(90, 93)),

    torqueMin: safeParse(clean.substring(116, 122), 100),
    torqueMax: safeParse(clean.substring(124, 130), 100),
    torqueTarget: safeParse(clean.substring(132, 138), 100),
    torqueValue: safeParse(clean.substring(140, 146), 100),

    angleMin: safeParse(clean.substring(148, 153)),
    angleMax: safeParse(clean.substring(155, 160)),
    angleTarget: safeParse(clean.substring(162, 167)),
    angleValue: safeParse(clean.substring(169, 174)),

    tighteningStatus: parseInt(clean.substring(107, 108)) || 0,
  };
}

module.exports = {
  buildFrame,
  buildMID0001,
  buildMID0050,
  buildMID0060,
  buildMID0062,
  buildMID9999,
  parseMID,
  parseMID0061,
  getReplyMID,
};
