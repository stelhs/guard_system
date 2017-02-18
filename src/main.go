package main 

import (
	"fmt"
	"mod_io"
	"huawei_e303"
	"guard_system"
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

const (
	DEV_FILE = "/dev/ttyS0"
	MODEM_IP = "192.168.1.1"
)



func main() {
	var err error
	
    db, err := sql.Open("mysql", "root:13941@/guard_system")
    if err != nil {
        panic(fmt.Sprintf("main: can't open mysql connection: %v", err))
    }
	defer db.Close()	

	modem := huawei_e303.New(MODEM_IP)
	mio, err := mod_io.New(DEV_FILE)
	if err != nil {
		panic(fmt.Sprintf("main: can't create mod_io: %v", err))
	}
	
	gs := guard_system.New(db, mio, modem)
	if err != nil {
		panic(fmt.Sprintf("main: can't create guard_system: %v", err))
	}
	
	fmt.Println("set GS to ready")
	
	gs.Guard_start("sms")
	
	for {
		msg := mio.Recv("AIP", 0)
		fmt.Println("recv msg = ", msg)
		if msg == nil {
			continue
		}
		
		if msg.Si == "AIP" {
			gs.Set_relay_state(3, msg.Args[1])
		}
	}
}

