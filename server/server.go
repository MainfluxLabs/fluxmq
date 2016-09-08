/**
 * Copyright (c) Mainflux
 *
 * FluxMQ is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package server

import (
	"fmt"
    "net"
    "os"
	"sync"
	"time"
	"strconv"
	"io/ioutil"
	"os/signal"
)


// Server is our main struct.
type Server struct {
	mu            sync.Mutex
	trace         bool
	debug         bool
	running       bool
	listener      net.Listener
	//clients       map[uint64]*client
	totalClients  uint64
	done          chan bool
	start         time.Time
	opts          *Options
}

// New will setup a new server struct after parsing the options.
func New(opts *Options) *Server {

	s := &Server{
		done:	make(chan bool, 1),
		start:	time.Now(),
		debug:	opts.Debug,
		trace:	opts.Trace,
		opts:	opts,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// For tracking clients
	//s.clients = make(map[uint64]*client)

	s.handleSignals()

	return s
}

// PrintAndDie is exported for access in other packages.
func PrintAndDie(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
	os.Exit(1)
}

// PrintServerAndExit will print our version and exit.
func PrintServerAndExit() {
	fmt.Printf("FluxMQ version %s\n", VERSION)
	os.Exit(0)
}

// Signal Handling
func (s *Server) handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			fmt.Println("Trapped Signal; %v", sig)
			fmt.Println("Server Exiting..")
			os.Exit(0)
		}
	}()
}

// Protected check on running state
func (s *Server) isRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Server) logPid() {
	pidStr := strconv.Itoa(os.Getpid())
	err := ioutil.WriteFile(s.opts.PidFile, []byte(pidStr), 0660)
	if err != nil {
		PrintAndDie(fmt.Sprintf("Could not write pidfile: %v\n", err))
	}
}

// Start up the server, this will block.
// Start via a Go routine if needed.
func (s *Server) Start() {
	fmt.Println("Starting FluxMQ version %s", VERSION)
	//fmt.Println("Go build version %s", s.info.GoVersion)

	// Avoid RACE between Start() and Shutdown()
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	// Log the pid to a file
	if s.opts.PidFile != _EMPTY_ {
		s.logPid()
	}

	// Wait for clients.
	s.AcceptLoop()
}

// AcceptLoop is exported for easier testing.
func (s *Server) AcceptLoop() {

	hp := net.JoinHostPort(s.opts.Host, strconv.Itoa(s.opts.Port))
	fmt.Println("Listening for client connections on %s", hp)

	l, e := net.Listen("tcp", hp)
	if e != nil {
		fmt.Println("Error listening on port: %s, %q", hp, e)
		return
	}

	println("Server is ready")

	// Setup state that can enable shutdown
	s.mu.Lock()
	s.listener = l

	// If server was started with RANDOM_PORT (-1), opts.Port would be equal
	// to 0 at the beginning this function. So we need to get the actual port
	if s.opts.Port == 0 {
		// Write resolved port back to options.
		_, port, err := net.SplitHostPort(l.Addr().String())
		if err != nil {
			fmt.Println("Error parsing server address (%s): %s", l.Addr().String(), e)
			s.mu.Unlock()
			return
		}
		portNum, err := strconv.Atoi(port)
		if err != nil {
			fmt.Println("Error parsing server address (%s): %s", l.Addr().String(), e)
			s.mu.Unlock()
			return
		}
		s.opts.Port = portNum
	}
	s.mu.Unlock()

	tmpDelay := ACCEPT_MIN_SLEEP

	for s.isRunning() {
		conn, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				fmt.Println("Temporary Client Accept Error(%v), sleeping %dms",
					ne, tmpDelay/time.Millisecond)
				time.Sleep(tmpDelay)
				tmpDelay *= 2
				if tmpDelay > ACCEPT_MAX_SLEEP {
					tmpDelay = ACCEPT_MAX_SLEEP
				}
			} else if s.isRunning() {
				fmt.Println("Accept error: %v", err)
			}
			continue
		}
		tmpDelay = ACCEPT_MIN_SLEEP

		fmt.Println(conn)

		//s.startGoRoutine(func() {
		//	s.createClient(conn)
		//	s.grWG.Done()
		//})
	}
	fmt.Println("Server Exiting..")
	s.done <- true
}

