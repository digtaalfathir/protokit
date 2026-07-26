# modbus (Go)

Modbus **TCP** client (master) — Go port of the Node
[@digta/modbus](../../node/modbus). No external dependencies.

Import: `github.com/digtaalfathir/protokit/go/modbus`

## Usage

```go
c, err := modbus.Connect("192.168.1.10:502", modbus.WithUnitID(1))
if err != nil {
    log.Fatal(err)
}
defer c.Close()

regs, _ := c.ReadHoldingRegisters(0, 10) // []uint16
_ = c.WriteSingleRegister(4, 1234)
coils, _ := c.ReadCoils(0, 8)             // []bool
```

Methods: `ReadCoils`, `ReadDiscreteInputs`, `ReadHoldingRegisters`,
`ReadInputRegisters`, `WriteSingleCoil`, `WriteSingleRegister`,
`WriteMultipleCoils`, `WriteMultipleRegisters`. Server exceptions come back as
`*ModbusError` (with `.Code`).
