/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package main

import (
	"strconv"
	"github.com/mainflux/fluxmq/config"
	"github.com/fatih/color"
)

func main() {
	// Parse config
	var cfg config.Config
	cfg.Parse()

	// Print banner
	color.Cyan(banner)
	color.Cyan("Magic happens on port " + strconv.Itoa(cfg.MqttPort))
}

var banner = `
███████╗██╗     ██╗   ██╗██╗  ██╗    ███╗   ███╗ ██████╗ 
██╔════╝██║     ██║   ██║╚██╗██╔╝    ████╗ ████║██╔═══██╗
█████╗  ██║     ██║   ██║ ╚███╔╝     ██╔████╔██║██║   ██║
██╔══╝  ██║     ██║   ██║ ██╔██╗     ██║╚██╔╝██║██║▄▄ ██║
██║     ███████╗╚██████╔╝██╔╝ ██╗    ██║ ╚═╝ ██║╚██████╔╝
╚═╝     ╚══════╝ ╚═════╝ ╚═╝  ╚═╝    ╚═╝     ╚═╝ ╚══▀▀═╝ 

               == Industrial MQTT Broker ==

               Made with <3 by Mainflux Team
[w] http://mainflux.io
[t] @mainflux
`
