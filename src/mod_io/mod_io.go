package mod_io

import (
	"fmt"
	"nmea0183"
	"os"
	"os/exec"
	"container/list"
	"sync"
	"time"
)

type Mod_io struct {
	sync.Mutex
	nmea *nmea0183.Nmea0183
	dev *os.File
	tx chan string
	rx_queue *list.List
	rx chan *nmea0183.Nmea_msg
}


func New(dev_file string) (*Mod_io, error) {
	var err error
	
	mio := new(Mod_io)
	mio.tx = make(chan string, 16)
	mio.rx = make(chan *nmea0183.Nmea_msg, 16)
	mio.rx_queue = list.New()
	
	mio.dev, err = os.OpenFile(dev_file, os.O_RDWR | os.O_APPEND, 0660)
	if err != nil {
		return nil, err
	}
	
	mio.nmea = nmea0183.New()
	
	err = exec.Command("bash", "-c", "stty -F" + dev_file + " 9600 raw -echo").Run()
	if err != nil {
		return nil, err
	}
	
	return mio, err
}


func (mio *Mod_io) Receiver_thread() {
	var buf [64]byte
	var err error
	var count int
	
	for {
		count, err = mio.dev.Read(buf[:])
		if err != nil {
			continue; // TODO:
		}
		
		if count <= 0 {
			continue; // TODO:
		}
		
		for _, byte := range buf[:count] {
			msg := mio.nmea.Push_rxb(byte)
			if msg == nil {
				continue	
			}

			mio.rx <- msg
		}
	}
}


func (mio *Mod_io) Transmitter_thread() {
	var count int

	for {
		msg := <- mio.tx
		count = 0
		for count < len(msg) {
			var err error
			fmt.Println("send", msg)
			count, err = mio.dev.Write([]byte(msg))
			if err != nil {
				panic("Can't write to UART")
			}
		}
	}
}

// Send nmea0183 message to transmitter
func (mio *Mod_io) Send_cmd(ti string, si string, args []uint) {
	msg := mio.nmea.Create_msg(ti, si, args)
	mio.tx <- msg
}

// Set outport new state 
func (mio *Mod_io) Relay_set_state(port_num uint, state uint) {
	for cnt := 0; cnt < 3; cnt++ {
		mio.Send_cmd("PC", "RWS", []uint{port_num, state})
		msg := mio.Recv("SOP", 300)
		if msg == nil {
			continue
		}
		
		if msg.Args[0] != port_num {
			continue
		}
		
		if msg.Args[1] != state {
			continue
		}
		
		return
	}
	panic("mod_io: can't set relay state")	
}


// Get input port state
func (mio *Mod_io) Get_input_port_state(port_num uint) uint {
	for cnt := 0; cnt < 3; cnt++ {
		mio.Send_cmd("PC", "RIP", []uint{port_num})
		msg := mio.Recv("AIP", 300)
		if msg == nil {
			continue
		}
		
		if msg.Args[0] != port_num {
			continue
		}
		
		return msg.Args[1]
	}
	panic("mod_io: can't get input state")
	return 0	
}


// Set WDT state
func (mio *Mod_io) Wdt_set_state(state uint) {
	for cnt := 0; cnt < 3; cnt++ {
		mio.Send_cmd("PC", "WDC", []uint{state})
		msg := mio.Recv("WDS", 300)
		if msg == nil {
			continue
		}
		
		if (msg.Args[0] & 1) != state {
			continue
		}
		
		return
	}
	panic("mod_io: can't set relay state")	
}


// WDT reset
func (mio *Mod_io) Wdt_reset(state uint) {
	mio.Send_cmd("PC", "WRS", []uint{})
}


// Receive nmea0183 message by mask
func (mio *Mod_io) Recv(si string, timeout uint) *nmea0183.Nmea_msg {
	mio.Lock()
	for e := mio.rx_queue.Front(); e != nil; e = e.Next() {
		msg, _ := e.Value.(*nmea0183.Nmea_msg)
		
		if len(si) == 0 {
			mio.rx_queue.Remove(e)
			mio.Unlock()
			return msg
		}
		
		if msg.Si == si {
			mio.rx_queue.Remove(e)
			mio.Unlock()
			return msg
		}
	}
	mio.Unlock()

	for {
		var msg *nmea0183.Nmea_msg = nil 
		
		if timeout > 0 {
			select {
			case msg = <- mio.rx:
				break
				
			case <- time.After(time.Millisecond * 
								time.Duration(timeout)):
				return nil
			}
		} else {
			msg = <- mio.rx
		}
		
		if msg == nil {
			return nil
		}
		
		if len(si) == 0 {
			return msg
		}

		if msg.Si == si {
			return msg
		}
		
		mio.Lock()
		mio.rx_queue.PushBack(msg)
		mio.Unlock()
	}

	return nil
}

