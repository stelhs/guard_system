package main 

import (
	"fmt"
	"os"
	"mod_io"
	"huawei_e303"
	"time"
)

const (
	DEV_FILE = "/dev/ttyS0"
)

func main() {
	var err error
	var text string
	
	m := huawei_e303.New("192.168.1.1")
	fmt.Println(m)
//	err := m.Send_sms("+375295051024", "bla bla")
	stat, err := m.Get_traffic_statistics()
	fmt.Printf("Modem error: %v\n", err)
	fmt.Printf("stat: %v\n", stat)
	return
	
	sms_list, err := m.Check_for_new_sms()
	fmt.Printf("Modem error: %v\n", err)
	fmt.Printf("count SMS: %d\n", len(sms_list))
	fmt.Printf("SMS list: %v\n", sms_list)
	
	for _, sms := range sms_list {
		fmt.Println("sms index = ", sms.Index)
		m.Remove_sms(sms.Index)
	}
	return
	
	
	err = m.Send_ussd("*100#")
	if err != nil {
		fmt.Printf("Modem error: %v\n", err)
	}
	
	text, err = m.Check_for_new_ussd()
	fmt.Println("check for ussd: ", text)
	time.Sleep(time.Second * 5)
	text, err = m.Check_for_new_ussd()
	fmt.Println("check for ussd: ", text)
	return

	mio, err := mod_io.New(DEV_FILE)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
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

