// Read/write a PLC over Modbus TCP using @digta/modbus.
const { ModbusTcpClient } = require("@digta/modbus");

const client = new ModbusTcpClient({
  host: "192.168.250.1", // PLC IP
  port: 502,             // standard Modbus TCP port
  unitId: 1,             // Modbus slave/unit id
});

async function run() {
  try {
    await client.connect();
    console.log("Connected to PLC");

    const regs = await client.readHoldingRegisters(0, 10); // 10 words from D0
    console.log("Holding registers:", regs);

    // await client.writeMultipleRegisters(0, [1, 2, 3, 4, 5]);
    // await client.writeSingleRegister(4, 1234);
  } catch (err) {
    console.error("Modbus error:", err.message);
  } finally {
    client.close();
  }
}

run();
