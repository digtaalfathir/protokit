const ModbusRTU = require("modbus-serial");
const client = new ModbusRTU();

// IP PLC
const PLC_IP = "192.168.250.1";
const MODBUS_PORT = 500;

// Koneksi ke PLC
async function connectPLC() {
  try {
    await client.connectTCP(PLC_IP, { port: MODBUS_PORT });
    console.log("Terhubung ke PLC Omron NX1P2");
    client.setID(1); // Modbus ID default
  } catch (error) {
    console.error("Gagal terhubung ke PLC:", error.message);
  }
}

// Membaca Holding Register
async function readHoldingRegisters() {
  try {
    const address = 0; // Ganti dengan alamat register yang sesuai
    const length = 10; // Jumlah register yang dibaca

    const data = await client.readHoldingRegisters(address, length);
    console.log("Data dari PLC:", data.data);
  } catch (error) {
    console.error("Gagal membaca register:", error.message);
  }
}

async function writeRegisters() {
  try {
    const address = 0; // Alamat register tujuan
    const values = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]; // Nilai yang dikirim ke PLC

    await client.writeRegisters(address, values);
    console.log("📤 Data berhasil ditulis ke PLC:", values);
  } catch (error) {
    console.error("❌ Gagal menulis register:", error.message);
  }
}

// Eksekusi Program
async function run() {
  await connectPLC();
  await readHoldingRegisters();
  //   await writeRegisters(); // Menulis data setelah membaca
  client.close();
}

run();
