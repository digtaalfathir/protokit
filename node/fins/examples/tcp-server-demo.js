const net = require("net");

const PORT = 900; // Sesuaikan dengan port yang digunakan di PLC
const HOST = "192.168.250.19";

const server = net.createServer((socket) => {
  console.log("Client connected:", socket.remoteAddress, socket.remotePort);

  // Menerima data dari PLC
  socket.on("data", (data) => {
    console.log("Received raw data:", data);

    // Konversi byte menjadi status bit individu
    const receivedByte = data[0];
    const bitArray = [];
    for (let i = 0; i < 3; i++) {
      bitArray.push((receivedByte >> i) & 1);
    }

    console.log("Received bit array:", bitArray);

    // Format hasil agar lebih jelas
    console.log(`PLC Inputs: 02 = ${bitArray[2]}, 01 = ${bitArray[1]}, 00 = ${bitArray[0]}`);

    // Kirim balasan ke PLC (sesuaikan dengan format yang dibutuhkan)
    socket.write("ACK");
  });

  socket.on("error", (err) => {
    console.error("Socket error:", err.message);
  });

  socket.on("close", () => {
    console.log("Client disconnected");
  });
});

server.listen(PORT, HOST, () => {
  console.log(`Server listening on ${HOST}:${PORT}`);
});
