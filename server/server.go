/**
 * Copyright (c) Mainflux
 *
 * FluxMQ is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package server

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"
)

// Info is the information sent to clients to help them understand information
// about this server.
type Info struct {
	ID                string   `json:"server_id"`
	Version           string   `json:"version"`
	GoVersion         string   `json:"go"`
	Host              string   `json:"host"`
	Port              int      `json:"port"`
	MaxPayload        int      `json:"max_payload"`
	IP                string   `json:"ip,omitempty"`
	ClientConnectURLs []string `json:"connect_urls,omitempty"` // Contains URLs a client can connect to.

	// Used internally for quick look-ups.
	clientConnectURLs map[string]struct{}
}

// Server is our main struct.
type Server struct {
	mu            sync.Mutex
	trace         bool
	debug         bool
	running       bool
	listener      net.Listener
	clients       map[uint64]*client
	routes        map[uint64]*client
	remotes       map[string]*client
	totalClients  uint64
	done          chan bool
	start         time.Time
	opts          *Options
	info          Info
	infoJSON      []byte
	grWG          sync.WaitGroup // to wait on various go routines
	routeInfo     Info
	routeInfoJSON []byte
	rcQuit        chan bool
	grMu          sync.Mutex
	grTmpClients  map[uint64]*client
	grRunning     bool
}

// Make sure all are 64bits for atomic use
type stats struct {
	inMsgs        int64
	outMsgs       int64
	inBytes       int64
	outBytes      int64
	slowConsumers int64
}

func (s *Server) getOpts() *Options {
	//s.optsMu.RLock()
	opts := s.opts
	//s.optsMu.RUnlock()
	return opts
}

func (s *Server) setOpts(opts *Options) {
	//s.optsMu.Lock()
	s.opts = opts
	//s.optsMu.Unlock()
}

func (s *Server) generateServerInfoJSON() {
	// Generate the info json
	b, err := json.Marshal(s.info)
	if err != nil {
		//s.Fatalf("Error marshaling INFO JSON: %+v\n", err)
		return
	}
	println(b)
	//s.infoJSON = []byte(fmt.Sprintf("INFO %s %s", b, CR_LF))
}

// New will setup a new server struct after parsing the options.
func New(opts *Options) *Server {

	s := &Server{
		done:  make(chan bool, 1),
		start: time.Now(),
		debug: opts.Debug,
		trace: opts.Trace,
		opts:  opts,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// For tracking clients
	s.clients = make(map[uint64]*client)

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

// Start up the server, this will block.
// Start via a Go routine if needed.
func (s *Server) Start() {
	fmt.Printf("Starting FluxMQ version %s\n", VERSION)
	//fmt.Println("Go build version %s", s.info.GoVersion)

	// Avoid RACE between Start() and Shutdown()
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	// Wait for clients.
	s.AcceptLoop()
}

// AcceptLoop is exported for easier testing.
func (s *Server) AcceptLoop() {

	hp := net.JoinHostPort(s.opts.Host, strconv.Itoa(s.opts.Port))
	fmt.Printf("Listening for client connections on %s\n", hp)

	l, e := net.Listen("tcp", hp)
	if e != nil {
		fmt.Printf("Error listening on port: %s, %q\n", hp, e)
		return
	}

	println("Server is ready")

	s.listener = l

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

		s.startGoRoutine(func() {
			s.createClient(conn)
			s.grWG.Done()
		})
	}
	fmt.Println("Server Exiting..")
	s.done <- true
}

// ID returns the server's ID
func (s *Server) ID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info.ID
}

func (s *Server) startGoRoutine(f func()) {
	s.grMu.Lock()
	if s.grRunning {
		s.grWG.Add(1)
		go f()
	}
	s.grMu.Unlock()
}

func (s *Server) createClient(conn net.Conn) *client {
	c := &client{srv: s, nc: conn, opts: defaultOpts, mpay: s.info.MaxPayload, start: time.Now()}

	// Grab JSON info string
	s.mu.Lock()
	info := s.infoJSON
	s.totalClients++
	s.mu.Unlock()

	// Grab lock
	c.mu.Lock()

	// Initialize
	c.initClient()

	// Send our information.
	c.sendInfo(info)

	// Unlock to register
	c.mu.Unlock()

	// Register with the server.
	s.mu.Lock()
	// If server is not running, Shutdown() may have already gathered the
	// list of connections to close. It won't contain this one, so we need
	// to bail out now otherwise the readLoop started down there would not
	// be interrupted.
	if !s.running {
		s.mu.Unlock()
		return c
	}

	// Snapshot server options.
	opts := s.getOpts()

	// If there is a max connections specified, check that adding
	// this new client would not push us over the max
	if opts.MaxConn > 0 && len(s.clients) >= opts.MaxConn {
		s.mu.Unlock()
		c.maxConnExceeded()
		return nil
	}
	s.clients[c.cid] = c
	s.mu.Unlock()

	// Do final client initialization

	// Set the Ping timer
	c.setPingTimer()

	// Spin up the read loop.
	s.startGoRoutine(func() { c.readLoop() })

	c.mu.Unlock()

	return c
}

/////////////////////////////////////////////////////////////////
// These are some helpers for accounting in functional tests.
/////////////////////////////////////////////////////////////////

// NumRoutes will report the number of registered routes.
func (s *Server) NumRoutes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.routes)
}

// NumRemotes will report number of registered remotes.
func (s *Server) NumRemotes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.remotes)
}

// NumClients will report the number of registered clients.
func (s *Server) NumClients() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}

// NumSubscriptions will report how many subscriptions are active.
func (s *Server) NumSubscriptions() uint32 {
	//s.mu.Lock()
	//subs := s.sl.Count()
	//s.mu.Unlock()
	//return subs
	return 0
}

// Addr will return the net.Addr object for the current listener.
func (s *Server) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// ReadyForConnections returns `true` if the server is ready to accept client
// and, if routing is enabled, route connections. If after the duration
// `dur` the server is still not ready, returns `false`.
func (s *Server) ReadyForConnections(dur time.Duration) bool {
	// Snapshot server options.
	//opts := s.getOpts()

	end := time.Now().Add(dur)
	for time.Now().Before(end) {
		s.mu.Lock()
		//ok := s.listener != nil && (opts.Cluster.Port == 0 || s.routeListener != nil)
		ok := s.listener != nil
		s.mu.Unlock()
		if ok {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}
