package conf

import (
	"fmt"
	"io/ioutil"
    "github.com/BurntSushi/toml"
)

type Main_config struct {
	Io_module Io_module_cfg
	Modem Modem_cfg
	Guard_settings Guard_sys_cfg
	Cameras map[string]Video_camera_cfg
}

type Guard_sys_cfg struct {
	List_phones []string
	Siren_io_port int
	Lamp_io_port int
	Sirena_timeout int
	Light_sensor_io_port int
	Light_ready_timeout int
	Light_sleep_timeout int
	Cameras *map[string]Video_camera_cfg
}

type Io_module_cfg struct {
	Uart_dev string
	Uart_speed string
	Responce_timeout int
	Repeate_count int
}

type Modem_cfg struct {
	Ip_addr string
}

type Video_camera_cfg struct {
	Id int
	V4l_dev string
	Resolution string
}

func Conf_parse() (*Main_config, error) {
	var err error
	var conf Main_config
	
	config_text, err := ioutil.ReadFile("/etc/guard_system.conf")
	if err != nil {
		return nil, fmt.Errorf("Can't open config file /etc/guard_system.conf: %v", err)
	}

	_, err = toml.Decode(string(config_text), &conf)
	if err != nil {
		return nil, fmt.Errorf("Can't decode config file /etc/guard_system.conf: %v", err)
	}
	
	conf.Guard_settings.Cameras = &conf.Cameras
	return &conf, nil
}

