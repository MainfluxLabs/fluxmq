/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	// MQTT
	MqttHost string
	MqttPort int
}


func (cfg *Config) Parse() {
	/**
	 * Config
	 */
	var confFile string

	testEnv := os.Getenv("TEST_ENV")
	if testEnv == "" && len(os.Args) > 1 {
		// We are not in the TEST_ENV (where different args are provided)
		// and provided config file as an argument
		confFile = os.Args[1]
	} else {
		confFile = "config/config.toml"
	}

	if _, err := toml.DecodeFile(confFile, &cfg); err != nil {
		// handle error
		fmt.Println("Error parsing Toml")
	}
}
