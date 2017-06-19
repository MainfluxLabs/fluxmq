package server

import (
	"bufio"
	"math/rand"
	"net"
	"sync"
	"time"
)

// Type of client connection.
const (
	// CLIENT is an end user.
	CLIENT = iota
	// ROUTER is another router in the cluster.
	ROUTER
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	// Scratch buffer size for the processMsg() calls.
	msgScratchSize = 512
	msgHeadProto   = "MSG "
)

// For controlling dynamic buffer sizes.
const (
	startBufSize = 512 // For INFO/CONNECT block
	minBufSize   = 128
	maxBufSize   = 65536
)

// Represent client booleans with a bitmask
type clientFlag byte

// Some client state represented as flags
const (
	connectReceived clientFlag = 1 << iota // The CONNECT proto has been received
	firstPongSent                          // The first PONG has been sent
	infoUpdated                            // The server's Info object has changed before first PONG was sent
)

// set the flag (would be equivalent to set the boolean to true)
func (cf *clientFlag) set(c clientFlag) {
	*cf |= c
}

// isSet returns true if the flag is set, false otherwise
func (cf clientFlag) isSet(c clientFlag) bool {
	return cf&c != 0
}

// setIfNotSet will set the flag `c` only if that flag was not already
// set and return true to indicate that the flag has been set. Returns
// false otherwise.
func (cf *clientFlag) setIfNotSet(c clientFlag) bool {
	if *cf&c == 0 {
		*cf |= c
		return true
	}
	return false
}

// clear unset the flag (would be equivalent to set the boolean to false)
func (cf *clientFlag) clear(c clientFlag) {
	*cf &= ^c
}

type client struct {
	// Here first because of use of atomics, and memory alignment.
	stats
	mu    sync.Mutex
	typ   int
	cid   uint64
	lang  string
	opts  clientOpts
	start time.Time
	nc    net.Conn
	mpay  int
	ncs   string
	bw    *bufio.Writer
	srv   *Server
	subs  map[string]*subscription
	perms *permissions
	cache readCache
	pcd   map[*client]struct{}
	atmr  *time.Timer
	ptmr  *time.Timer
	pout  int
	wfc   int
	msgb  [msgScratchSize]byte
	last  time.Time
	debug bool
	trace bool

	flags clientFlag // Compact booleans into a single field. Size will be increased when needed.
}

type permissions struct {
	pcache map[string]bool
}

const (
	maxResultCacheSize = 512
	maxPermCacheSize   = 32
	pruneSize          = 16
)

// Used in readloop to cache hot subject lookups and group statistics.
type readCache struct {
	genid   uint64
	prand   *rand.Rand
	inMsgs  int
	inBytes int
	subs    int
}

func (c *client) String() (id string) {
	return c.ncs
}

func (c *client) GetOpts() *clientOpts {
	return &c.opts
}

type subscription struct {
	client  *client
	subject []byte
	queue   []byte
	sid     []byte
	nm      int64
	max     int64
}

type clientOpts struct {
	Verbose       bool   `json:"verbose"`
	Pedantic      bool   `json:"pedantic"`
	SslRequired   bool   `json:"ssl_required"`
	Authorization string `json:"auth_token"`
	Username      string `json:"user"`
	Password      string `json:"pass"`
	Name          string `json:"name"`
	Lang          string `json:"lang"`
	Version       string `json:"version"`
	Protocol      int    `json:"protocol"`
}

var defaultOpts = clientOpts{Verbose: true, Pedantic: true}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Lock should be held
func (c *client) initClient() {

}

// Assume the lock is held upon entry.
func (c *client) sendInfo(info []byte) {
}

func (c *client) maxConnExceeded() {
	//c.Errorf(ErrTooManyConnections.Error())
	//c.sendErr(ErrTooManyConnections.Error())
	//c.closeConnection()
}

func (c *client) setPingTimer() {
	if c.srv == nil {
		return
	}
	//d := c.srv.getOpts().PingInterval
	//c.ptmr = time.AfterFunc(d, c.processPingTimer)
}

func (c *client) readLoop() {
}
