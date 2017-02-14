package main 

import (
	"fmt"
	"mod_io"
	"os"
)

const (
	DEV_FILE = "/dev/ttyUSB0"
)

func main() {
	mio, err := mod_io.New(DEV_FILE)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	
	go mio.Receiver_thread()
	go mio.Transmitter_thread()
	for {
		msg := mio.Recv("", 0)
		fmt.Println("recv msg = ", msg)
		if msg == nil {
			continue
		}
		
		if msg.Si == "AIP" {
			mio.Relay_set_state(3, msg.Args[1])
//			mio.Send_cmd("PC", "RWS", []uint{3, msg.Args[1]})
		}
	}
}

