package guard_system

import (
    "fmt"
    "time"
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "strings"
    "mod_io"
    "huawei_e303"
    "pthread"
    "conf"
)



// Sensor info
type sensor struct {
	id int
	name string
	port int
	normal_state int
}

// siren state in siren sequencer
type siren_state struct {
	state int
	interval uint
}

// The main Guard System descriptor
type Guard_system struct {
	cfg *conf.Guard_sys_cfg
	db *sql.DB
	mio *mod_io.Mod_io
	sensors_by_port map[int]sensor
	sensors_by_id map[int]sensor
	siren_threads []pthread.Thread
}



func New(cfg *conf.Guard_sys_cfg, db *sql.DB, 
			mio *mod_io.Mod_io, modem *huawei_e303.Modem) *Guard_system {
    var gs Guard_system
    gs.db = db
    gs.mio = mio
    gs.sensors_by_port = make(map[int]sensor)
    gs.sensors_by_id = make(map[int]sensor)
    gs.cfg = cfg
    
    // Get all sensors info
	rows, err := db.Query("SELECT id, name, " +
								"port, normal_state FROM sensors")
	if err != nil {
		panic(fmt.Sprintf("guard_system: can't create Guard_system object: %v", err))
	}

	for i := 0; rows.Next(); i++ {
		var sens sensor
	    err := rows.Scan(&sens.id, &sens.name, 
				    	 &sens.port, &sens.normal_state);
	    if  err != nil {
	        panic(fmt.Sprintf("guard_system: can't getting responce from DB: %v", err))
	    }
	    gs.sensors_by_port[sens.port] = sens
	    gs.sensors_by_id[sens.id] = sens
	}
	
    return &gs
}


func array_int_to_string(list []int) string {
	var part func ([]int) string
	if len(list) == 0 {
		return ""
	}

	part = func (list []int) string {
		if len(list) == 0 {
			return ""
		}
		return fmt.Sprintf(",%d", list[0]) + part(list[1:]);
	}
	return fmt.Sprintf("%d", list[0]) + part(list[1:]) 
}


func string_to_array_int(str string) []int {
	parts := strings.Split(str, ",")
	var arr []int
	for _, part := range parts {
		var val int
		fmt.Sscanf(part, "%d", &val)
		arr = append(arr, val)
	}
	return arr
}

func (gs *Guard_system) get_active_sensors() map[int]sensor {
	sensors := make(map[int]sensor)
	for port, sensor := range gs.sensors_by_port {
		var mode sql.NullString
		err := gs.db.QueryRow("SELECT mode FROM blocking_sensors " +
							"WHERE sense_id = ? " +
							"ORDER by created DESC " +
							"LIMIT 1", sensor.id).Scan(&mode)
		if err != nil && err != sql.ErrNoRows {
			panic(fmt.Sprintf("guard_system: can't get blocking_sensors state: %v", err))
		}
		
		if mode.Valid && mode.String == "lock" {
			continue
		}
		sensors[port] = sensor
	}
	return sensors
}


func (gs *Guard_system) Set_relay_state(port int, new_state int) {
	_, err := gs.db.Exec("insert into io_output_actions " + 
							"(port, state) " +
							"values (?, ?)", port, new_state)
	if err != nil {
		panic(fmt.Sprintf("guard_system: can't set_relay_state: %v", err))
	}
	
	gs.mio.Relay_set_state(port, new_state)
}


func (gs *Guard_system) Guard_start(method string) {
    // check for incorrect sensor value state
    // making ignore_sensors
	var ignore_sensors []int
	sensors := gs.get_active_sensors()
	for port, sensor := range sensors {
		curr_state := gs.mio.Get_input_port_state(port)
		if curr_state != sensor.normal_state {
			ignore_sensors = append(ignore_sensors, sensor.id)
		}
	}
	
	// set new guard state
	_, err := gs.db.Exec("insert into guard_states " + 
							"(state, method, ignore_sensors) " +
							"values ('ready', ?, ?)", method, 
								array_int_to_string(ignore_sensors))
	if err != nil {
		panic(fmt.Sprintf("guard_system: can't Guard_start: %v", err))
	}
	
	// indicate new guard state by siren
	if len(ignore_sensors) == 0 {
		gs.siren_start([]siren_state{{state: 1, interval: 200},
								 	 {state: 0, interval: 0}})
	} else {
		gs.siren_start([]siren_state{{state: 1, interval: 200},
									 {state: 0, interval: 200},
									 {state: 1, interval: 1000},
									 {state: 0, interval: 0},})
	}
	
	// send SMS
	gs.send_sms("guard_start", ignore_sensors)
}


func (gs *Guard_system) Guard_stop(method string) {
	// set new guard state
	_, err := gs.db.Exec("insert into guard_states " + 
							"(state, method) " +
							"values ('sleep', ?)", method)
	if err != nil {
		panic(fmt.Sprintf("guard_system: can't Guard_stop: %v", err))
	}
	
	// indicate new guard state by siren
	gs.siren_start([]siren_state{{state: 1, interval: 200},
								 {state: 0, interval: 200},
								 {state: 1, interval: 200},
								 {state: 0, interval: 0},})
	
	// send SMS
	//TODO: gs.send_sms("guard_stop")
}


func (gs *Guard_system) send_sms(sms_name string, args interface{}) {
	var sms_text string
	
	switch sms_name {
	case "guard_start":
		ignore_sensors := args.([]int)
		sms_text = fmt.Sprintf("Охрана включена.")
		if len(ignore_sensors) == 0 {
			break
		}
		
		sms_text += " Игнор:"
		separtor := ""
		for _, sens_id := range ignore_sensors {
			sms_text += separtor + " \"" + gs.sensors_by_id[sens_id].name + "\""
			separtor = ","
		}
		
	case "guard_stop":	
	}
	
	fmt.Println("sms_text = ", sms_text)
}



func (gs *Guard_system) siren_start(sequencer []siren_state) {
	gs.siren_stop()
	thread := pthread.Create(func () {
		for _, step := range sequencer {
			fmt.Println("gs.cfg.Siren_io_port = ", gs.cfg.Siren_io_port)
			gs.Set_relay_state(gs.cfg.Siren_io_port, step.state)
			if step.interval > 0 {
				time.Sleep(time.Millisecond * time.Duration(step.interval))
			}
		}
	})
	gs.siren_threads = append(gs.siren_threads, thread)
}

func (gs *Guard_system) siren_stop() {
	if len(gs.siren_threads) == 0 {
		return 
	}	

	for i, _ := range gs.siren_threads {
		gs.siren_threads[i].Kill()
	}
	gs.siren_threads = []pthread.Thread{}
	time.Sleep(time.Millisecond * 10)
	gs.Set_relay_state(gs.cfg.Siren_io_port, 0)
}

func (gs *Guard_system) Guard_get_state() (string, []int, error) {
	state := ""
	ignore_sensors_str := ""
	err := gs.db.QueryRow("SELECT state, ignore_sensors FROM" +
							" guard_states ORDER by created" +
							" DESC LIMIT 1").Scan(&state, &ignore_sensors_str)
	if err != nil {
		return "", nil, err
	}
	return state, string_to_array_int(ignore_sensors_str), nil
}





