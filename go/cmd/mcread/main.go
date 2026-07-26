// Command mcread reads D registers from a MELSEC PLC over MC protocol (3E).
//
//	go run ./cmd/mcread -addr 192.168.1.10:5007 -start 0 -count 10
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/digtaalfathir/protokit/go/mcprotocol"
)

func main() {
	addr := flag.String("addr", "192.168.1.10:5007", "PLC address host:port")
	start := flag.Uint("start", 0, "first D register")
	count := flag.Uint("count", 10, "number of words to read")
	flag.Parse()

	client, err := mcprotocol.Connect(*addr)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Close()

	words, err := client.ReadWords(mcprotocol.D, uint32(*start), uint16(*count))
	if err != nil {
		log.Fatalf("read: %v", err)
	}
	for i, w := range words {
		fmt.Printf("D%d = %d\n", *start+uint(i), w)
	}
}
