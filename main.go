/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/mainflux/fluxmq/server"
	"os"
	"strings"
)

var usageStr = `
Usage: fluxmq [options]
Server Options:
    -a, --addr <host>               Bind to host address (default: 0.0.0.0)
    -p, --port <port>               Use port for clients (default: 4222)
Logging Options:
    -l, --log <file>                File to redirect log output
Common Options:
    -h, --help                      Show this message
    -v, --version                   Show version
`

// usage will print out the flag options for the server.
func usage() {
	fmt.Printf("%s\n", usageStr)
	os.Exit(0)
}

func main() {
	// Server Options
	opts := server.Options{}

	var showVersion bool
	var debugAndTrace bool

	// Parse flags
	flag.IntVar(&opts.Port, "port", 1883, "Port to listen on.")
	flag.IntVar(&opts.Port, "p", 1883, "Port to listen on.")
	flag.StringVar(&opts.Host, "addr", "0.0.0.0", "Network host to listen on.")
	flag.StringVar(&opts.Host, "a", "0.0.0.0", "Network host to listen on.")
	flag.StringVar(&opts.LogFile, "l", "", "File to store logging output.")
	flag.StringVar(&opts.LogFile, "log", "", "File to store logging output.")
	flag.BoolVar(&showVersion, "version", false, "Print version information.")
	flag.BoolVar(&showVersion, "v", false, "Print version information.")

	flag.Usage = usage

	flag.Parse()

	// Show version and exit
	if showVersion {
		server.PrintServerAndExit()
	}

	// One flag can set multiple options.
	if debugAndTrace {
		opts.Trace, opts.Debug = true, true
	}

	// Process args looking for non-flag options,
	// 'version' and 'help' only for now
	for _, arg := range flag.Args() {
		switch strings.ToLower(arg) {
		case "version":
			server.PrintServerAndExit()
		case "help":
			usage()
		}
	}

	// Print banner
	color.Cyan(banner)

	// Create the server with appropriate options.
	s := server.New(&opts)

	// Start things up. Block here until done.
	s.Start()
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
