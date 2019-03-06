package halfpike

import (
	"context"
	"net"
	"time"
	"strings"
	"fmt"
	"strconv"

	"github.com/kylelemons/godebug/pretty"
)

func Example_long() {
	// A slice of structs that has a .Validate() method on it.
	// This is where our data will be stored.
	neighbors := BGPNeighbors{}

	// Creates our parers object that our various ParseFn functions will use to move
	// through the input.
	p, err := NewParser(showBGPNeighbor, neighbors)
	if err != nil {
		panic(err)
	}

	// An object that contains various ParseFn methods.
	states := &PeerParser{}

	// Parses our content in showBGPNeighbor and begins parsing with states.FindPeer
	// which is a ParseFn.
	if err := Parse(context.Background(), p, states.FindPeer); err != nil {
		panic(err)
	}

	// Because we pass in a slice, we have to do a reassign to get the changed value.
	neighbors = p.Validator.(BGPNeighbors) 
	fmt.Println(pretty.Sprint(neighbors))

/* Output:
[{PeerIP:     10.10.10.2,
  PeerPort:   179,
  PeerAS:     22,
  LocalIP:    10.10.10.1,
  LocalPort:  65406,
  LocalAS:    22,
  Type:       1,
  State:      3,
  LastState:  5,
  HoldTime:   90000000000,
  Preference: 170,
  PeerID:     10.10.10.2,
  LocalID:    10.10.10.1,
  InetStats:  {0: {ID:                 0,
                   Bit:                10000,
                   RIBState:           2,
                   SendState:          1,
                   ActivePrefixes:     0,
                   RecvPrefixes:       0,
                   AcceptPrefixes:     0,
                   SurpressedPrefixes: 2,
                   AdvertisedPrefixes: 0}}},
 {PeerIP:     10.10.10.6,
  PeerPort:   54781,
  PeerAS:     22,
  LocalIP:    10.10.10.5,
  LocalPort:  179,
  LocalAS:    22,
  Type:       1,
  State:      3,
  LastState:  5,
  HoldTime:   90000000000,
  Preference: 170,
  PeerID:     10.10.10.6,
  LocalID:    10.10.10.1,
  InetStats:  {0: {ID:                 0,
                   Bit:                10000,
                   RIBState:           2,
                   SendState:          1,
                   ActivePrefixes:     0,
                   RecvPrefixes:       0,
                   AcceptPrefixes:     0,
                   SurpressedPrefixes: 0,
                   AdvertisedPrefixes: 0}}}]
*/
}

// showBGPNeighbor is the output we are going to lex/parse.
var showBGPNeighbor = `
Peer: 10.10.10.2+179 AS 22     Local: 10.10.10.1+65406 AS 17   
  Type: External    State: Established    Flags: <Sync>
  Last State: OpenConfirm   Last Event: RecvKeepAlive
  Last Error: None
  Options: <Preference PeerAS Refresh>
  Holdtime: 90 Preference: 170
  Number of flaps: 0
  Peer ID: 10.10.10.2       Local ID: 10.10.10.1       Active Holdtime: 90
  Keepalive Interval: 30         Peer index: 0   
  BFD: disabled, down
  Local Interface: ge-1/2/0.0                       
  NLRI for restart configured on peer: inet-unicast
  NLRI advertised by peer: inet-unicast
  NLRI for this session: inet-unicast
  Peer supports Refresh capability (2)
  Restart time configured on the peer: 120
  Stale routes from peer are kept for: 300
  Restart time requested by this peer: 120
  NLRI that peer supports restart for: inet-unicast
  NLRI that restart is negotiated for: inet-unicast
  NLRI of received end-of-rib markers: inet-unicast
  NLRI of all end-of-rib markers sent: inet-unicast
  Peer supports 4 byte AS extension (peer-as 22)
  Peer does not support Addpath
  Table inet.0 Bit: 10000
    RIB State: BGP restart is complete
    Send state: in sync
    Active prefixes:              0
    Received prefixes:            0
    Accepted prefixes:            0
    Suppressed due to damping:    2
    Advertised prefixes:          0
  Last traffic (seconds): Received 10   Sent 6    Checked 1   
  Input messages:  Total 8522   Updates 1       Refreshes 0     Octets 161922
  Output messages: Total 8433   Updates 0       Refreshes 0     Octets 160290
  Output Queue[0]: 0

Peer: 10.10.10.6+54781 AS 22   Local: 10.10.10.5+179 AS 17   
  Type: External    State: Established    Flags: <Sync>
  Last State: OpenConfirm   Last Event: RecvKeepAlive
  Last Error: None
  Options: <Preference PeerAS Refresh>
  Holdtime: 90 Preference: 170
  Number of flaps: 0
  Peer ID: 10.10.10.6       Local ID: 10.10.10.1       Active Holdtime: 90
  Keepalive Interval: 30         Peer index: 1   
  BFD: disabled, down                   
  Local Interface: ge-0/0/1.5                       
  NLRI for restart configured on peer: inet-unicast
  NLRI advertised by peer: inet-unicast
  NLRI for this session: inet-unicast
  Peer supports Refresh capability (2)
  Restart time configured on the peer: 120
  Stale routes from peer are kept for: 300
  Restart time requested by this peer: 120
  NLRI that peer supports restart for: inet-unicast
  NLRI that restart is negotiated for: inet-unicast
  NLRI of received end-of-rib markers: inet-unicast
  NLRI of all end-of-rib markers sent: inet-unicast
  Peer supports 4 byte AS extension (peer-as 22)
  Peer does not support Addpath
  Table inet.0 Bit: 10000
    RIB State: BGP restart is complete
    Send state: in sync
    Active prefixes:              0
    Received prefixes:            0
    Accepted prefixes:            0
    Suppressed due to damping:    0
    Advertised prefixes:          0
  Last traffic (seconds): Received 12   Sent 6    Checked 33  
  Input messages:  Total 8527   Updates 1       Refreshes 0     Octets 162057
  Output messages: Total 8430   Updates 0       Refreshes 0     Octets 160233
  Output Queue[0]: 0
 `

// PeerType is the type of peer the neighbor is.
type PeerType uint8

// BGP neighbor types.
 const (
 	// PTUnknown indicates the neighbor type is unknown.
 	PTUnknown PeerType = 0
 	// PTExternal indicates the neighbor is external to the router's AS.
 	PTExternal PeerType = 1
 	// PTInternal indicates the neighbort is intneral to the router's AS.
 	PTInternal PeerType = 2
 )

 type BGPState uint8

 // BGP connection states.
 const(
 	NSUnknown BGPState = 0
 	NSActive BGPState = 1
 	NSConnect BGPState = 2
 	NSEstablished BGPState = 3
 	NSIdle BGPState = 4
 	NSOpenConfirm BGPState = 5
 	NSOpenSent BGPState = 6
 	NSRRClient BGPState = 7
 )

 type RIBState uint8

 const (
 	RSUnknown RIBState = 0
 	RSComplete RIBState =2
 	RSInProgress RIBState = 3
 )

 type SendState uint8

 const (
 	RSSendUnknown SendState = 0
 	RSSendSync SendState = 1
 	RSSendNotSync SendState = 2
 	RSSendNoAdvertise SendState = 3
 )

 // BGPNeighbors is a collection of BGPNeighbors for a router.
type BGPNeighbors []*BGPNeighbor

// Vaildate implements Validator.Validate(). 
func (b BGPNeighbors) Validate() error {
	for _, v := range b {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// BGPNeighbor provides information about a router's BGP Neighbor.
type BGPNeighbor struct {
	// PeerIP is the IP address of the neighbor.
	PeerIP net.IP
	// PeerPort is the IP port of the peer.
	PeerPort uint32
	// PeerAS is the peers autonomous system number.
	PeerAS int
	// LocalIP is the IP address on this router the neighbor connects to.
	LocalIP net.IP
	// LocaPort is the IP port on this router the neighbor connects to.
	LocalPort uint32
	// LocalAS is the local autonomous system number.
	LocalAS int
	// Type is the type of peer.
	Type PeerType
	// State is the current state of the BGP peer.
	State BGPState
	// LastState is the previous state of the BGP peer.
	LastState BGPState
	// HoldTime is how long to consider the neighbor valid after not hearing a keep alive.
	HoldTime time.Duration
	// Preference is the BGP preference value.
	Preference int
	// PeerID is the ID the peer uses to identify itself.
	PeerID net.IP
	// LocalID is the ID the local router uses to identify itself.
	LocalID net.IP
	InetStats map[int]*InetStats

	initCalled bool
}

func (b *BGPNeighbor) init() {
	b.Preference = -1
	b.initCalled = true
	b.LocalAS, b.PeerAS = -1, -1
}

// Vaildate implements Validator.Validate(). 
func (b *BGPNeighbor) Validate() error {
	if !b.initCalled {
		return fmt.Errorf("internal error: BGPNeighbor.init() was not called")
	} 
	
	switch {
	case b.PeerIP == nil:
		return fmt.Errorf("PeerIP was nil")
	case b.LocalIP == nil:
		return fmt.Errorf("LocalIP was nil")
	case b.PeerID == nil:
		return fmt.Errorf("PeerID was nil")
	case b.LocalID == nil:
		return fmt.Errorf("LocalID was nil")
	}

	switch uint32(0) {
	case b.PeerPort:
		return fmt.Errorf("PeerPort was 0")
	case b.LocalPort:
		return fmt.Errorf("LocalPort was 0")
	}

	switch 0 {
	case int(b.Type):
		return fmt.Errorf("Type was not set")
	case int(b.LastState):
		return fmt.Errorf("LastState was not set")
	case int(b.State):
		return fmt.Errorf("State was not set")
	}

	switch -1 {
	case b.Preference:
		return fmt.Errorf("Preference was not set")
	case b.LocalAS:
		return fmt.Errorf("LocalAS was not set")
	case b.PeerAS:
		return fmt.Errorf("PeerAS was not set")
	}

	for _, v := range b.InetStats {
		if err := v.Validate(); err != nil {
			err = fmt.Errorf(err.Error())
		}
	}
	return nil
}

// InetStats contains information about the route table.
type InetStats struct {
	ID int
	Bit int
	RIBState RIBState
	SendState SendState
	ActivePrefixes int
	RecvPrefixes int
	AcceptPrefixes int
	SurpressedPrefixes int
	AdvertisedPrefixes int
}

func (b *InetStats) init() {
	b.Bit = -1
	b.ActivePrefixes = -1
	b.RecvPrefixes = -1
	b.AcceptPrefixes = -1
	b.SurpressedPrefixes = -1
}

// Validate implements Validator.
func (b *InetStats) Validate() error {
	switch -1 {
	case b.Bit:
		return fmt.Errorf("InetStats: Bit was not parsed from the input")
	case b.ActivePrefixes:
		return fmt.Errorf("InetStats(Bit==%d): ActivePrefixes was not parsed from the input", b.Bit)
	case b.AcceptPrefixes:
		return fmt.Errorf("InetStats(Bit==%d): AcceptPrefixes was not parsed from the input", b.Bit)
	case b.SurpressedPrefixes:
		return fmt.Errorf("InetStats(Bit==%d): SurpressedPrefixes was not parsed from the input", b.Bit)
	}

	switch {
	case b.RIBState == RSUnknown:
		return fmt.Errorf("InetStats(Bit==%d): RIBState was unknown, which indicates the parser is broken on input", b.Bit)
	case b.SendState == RSSendUnknown:
		return fmt.Errorf("InetStats(Bit==%d): SendState was unknown, which indicates the parser is broken on input", b.Bit)
	}
	return nil
}

// PeerParse is a collection of ParseFn for parsing top level entries in "show bgp neigbhors".
type PeerParser struct {
	parser *Parser
	peers BGPNeighbors
}

// lastPeer returns the last *BgPNeighbor added to our *BGPNeighbors slice.
func (pe *PeerParser) lastPeer() *BGPNeighbor {
	if len(pe.peers) == 0 {
		return nil
	}
	return pe.peers[len(pe.peers)-1]
}

func (pe *PeerParser) errorf(s string, a ...interface{}) ParseFn {
	rec := pe.lastPeer()
	if rec == nil {
		return pe.parser.Errorf(s, a...)
	}

	return pe.parser.Errorf("Peer(%s+%d):Local(%s+%d) entry: %s", rec.PeerIP, rec.PeerPort, rec.LocalIP, rec.LocalPort, fmt.Sprintf(s, a...))
}

var peerRecStart = []string{"Peer:", Skip, "AS", Skip, "Local:", Skip, "AS", Skip}

// Peer: 10.10.10.2+179 AS 22     Local: 10.10.10.1+65406 AS 17 
func (pe *PeerParser) FindPeer(ctx context.Context, p *Parser) ParseFn {
	const (
		peerIPPort = 1
		peerASNum = 3 
		localIPPort = 5
		localASNum = 7
	)
	pe.peers = p.Validator.(BGPNeighbors)
	pe.parser = p

	rec := &BGPNeighbor{}
	rec.init()

	line, err := p.FindStart(peerRecStart)
	if err != nil {
		if len(pe.peers) == 0 {
			p.Errorf("did not locate the start of our list of peers within the output")
		}
		return nil
	}
	if p.EOF(line) {
		return p.Errorf("received the start of a Peer statement, but then an EOF: %#+v", line)
	}

	// Get peer's ip and port.
	ip, port, err := ipPort(line.Items[peerIPPort].Val)
	if err != nil {
		return p.Errorf("coud not retrieve a valid peer IP and port from: %#+v", line)
	}
	rec.PeerIP = ip
	rec.PeerPort = uint32(port)

	// Get peer's AS.
	as, err :=line.Items[peerASNum].ToInt()
	if err != nil {
		return p.Errorf("could not retrieve the peer AS num from: %#+v", line)
	}
	rec.PeerAS = as

	// Get local ip and port.
	ip, port, err = ipPort(line.Items[localIPPort].Val)
	if err != nil {
		return p.Errorf("coud not retrieve a valid local IP and port from: %#+v", line)
	}
	rec.LocalIP = ip
	rec.LocalPort = uint32(port)

	// Get local AS.
	as, err = line.Items[peerASNum].ToInt()
	if err != nil {
		return p.Errorf("could not retrieve the peer AS num from: %#+v", line)
	}
	rec.LocalAS = as

	pe.peers = append(pe.peers, rec)
	return pe.typeState
}

var toPeerType = map[string]PeerType {
	"Internal": PTInternal,
	"External": PTExternal,
}

var toState = map[string]BGPState{
	"Active": NSActive,
 	"Connect": NSConnect,
 	"Established": NSEstablished,
 	"Idle": NSIdle,
 	"OpenConfirm": NSOpenConfirm,
 	"OpenSent": NSOpenSent,
 	"route reflector client": NSRRClient,
}

// Type: External    State: Established    Flags: <Sync>
func (pe *PeerParser) typeState(ctx context.Context, p *Parser) ParseFn {
	const (
		peerType = 1
		state = 3
	)

	line := p.Next()

	rec := pe.lastPeer()
	
	if !p.IsAtStart(line, []string{"Type:", Skip, "State:", Skip}) {
		return pe.errorf("did not have the expected 'Type' and 'State' declarations following peer line")
	}

	t, ok := toPeerType[line.Items[peerType].Val]
	if !ok {
		return pe.errorf("Type was not 'Internal' or 'External', was %s", line.Items[peerType].Val)
	}
	rec.Type = t

	s, ok := toState[line.Items[state].Val]
	if !ok {
		return pe.errorf("BGP State was not one of the accepted types (Active, Connect, ...), was %s", line.Items[state].Val)
	}
	rec.State = s

	return pe.lastState
}

// Last State: OpenConfirm   Last Event: RecvKeepAlive
func (pe *PeerParser) lastState(ctx context.Context, p *Parser) ParseFn {
	line := p.Next()

	rec := pe.lastPeer()

	if !p.IsAtStart(line, []string{"Last", "State:", Skip}) {
		return pe.errorf("did not have the expected 'Last State:', got %#+v", line)
	}

	s, ok := toState[line.Items[2].Val]
	if !ok {
		return pe.errorf("BGP last state was not one of the accepted types (Active, Connect, ...), was %s", line.Items[2].Val)
	}
	rec.LastState = s
	return pe.holdTimePref
}

// Holdtime: 90 Preference: 170
func (pe *PeerParser) holdTimePref(ctx context.Context, p *Parser) ParseFn {
	const (
		hold = 1
		pref = 3
	)

	rec := pe.lastPeer()

	line, until, err := p.FindUntil([]string{"Holdtime:", Skip, "Preference:", Skip}, peerRecStart)
	if err != nil {
		return pe.errorf("reached end of file before finding Holdtime and Preference line")
	}
	if until {
		return pe.errorf("reached next entry before finding Holdtime and Preference line")
	}

	ht, err := line.Items[hold].ToInt()
	if err != nil {
		return pe.errorf("Holdtime was not an integer, was %s",line.Items[hold].Val)
	}
	prefVal, err := line.Items[pref].ToInt()
	if err != nil {

		return pe.errorf("Preference was not an integer, was %s", line.Items[pref].Val)
	}

	rec.HoldTime = time.Duration(ht) * time.Second
	rec.Preference = prefVal
	return pe.peerIDLocalID
}

// Peer ID: 10.10.10.6       Local ID: 10.10.10.1       Active Holdtime: 90
func (pe *PeerParser) peerIDLocalID(ctx context.Context, p *Parser) ParseFn {
	const (
		peer = 2
		local = 5
	)

	rec := pe.lastPeer()

	line, until, err := p.FindUntil([]string{"Peer", "ID:", Skip, "Local", "ID:", Skip}, peerRecStart)
	if err != nil {
		return pe.errorf("reached end of file before finding PeerID and LocalID")
	}
	if until {
		return pe.errorf("reached next entry before finding PeerID and LocalID")
	}
	pid := net.ParseIP(line.Items[peer].Val)
	if pid == nil {
		return pe.errorf("PeerID does not appear to be an IP: was %s", line.Items[peer].Val)
	}
	loc := net.ParseIP(line.Items[local].Val)
	if loc == nil {
		return pe.errorf("LocalID does not appear to be an IP: was %s", line.Items[local].Val)
	}
	rec.PeerID = pid
	rec.LocalID = loc
	return pe.findTableStats	
}

// Table inet.0 Bit: 10000
func (pe *PeerParser) findTableStats(ctx context.Context, p *Parser) ParseFn { 
	p.Validator = pe.peers

	_, until, err := p.FindUntil([]string{"Table", Skip, "Bit:", Skip}, peerRecStart)
	if err != nil {
		return nil
	}
	if until{
		return pe.FindPeer
	}
	p.Backup()
	ts := &tableStats{peer: pe}
	return ts.start
}

/*
Table inet.0 Bit: 10000
    RIB State: BGP restart is complete
    Send state: in sync
    Active prefixes:              0
    Received prefixes:            0
    Accepted prefixes:            0
    Suppressed due to damping:    0
    Advertised prefixes:          0
*/
type tableStats struct{
	peer *PeerParser
	stats *InetStats
	rec *BGPNeighbor
}

func (t *tableStats) errorf(s string, a ...interface{}) ParseFn {
	if t.stats == nil {
		return t.peer.errorf("Table(unknown): %s", fmt.Sprintf(s, a...))
	}
	return t.peer.errorf("Table(ID: %d, Bit: %d): %s", t.stats.ID, t.stats.Bit, fmt.Sprintf(s, a...))
}

// Table inet.0 Bit: 10000
func (t *tableStats) start(ctx context.Context, p *Parser) ParseFn {
	const (
		table = 1
		bit = 3
	)
	t.rec = t.peer.lastPeer()

	line := p.Next()

	tvals := strings.Split(line.Items[table].Val, `.`)
	if len(tvals) != 2 {
		return t.errorf("had Table entry with table id that wasn't in a format I understand: %s", line.Items[table].Val)
	}
	i, err := strconv.Atoi(tvals[1])
	if err != nil {
		return t.errorf("had Table entry with table id that wasn't an integer: %s", tvals[1])
	}

	b, err := line.Items[bit].ToInt()
	if err != nil {
		return t.errorf("had Table entry with bits id that wasn't an integer: %s", line.Items[bit].Val)
		return nil
	}
	t.stats = &InetStats{
		ID: i,
		Bit: b,
	}

	return t.ribState
}

var toRIBState = map[string]RIBState {
	"restart is complete": RSComplete,
	"estart in progress": RSInProgress, 
}


// RIB State: BGP restart is complete
func (t *tableStats) ribState(ctx context.Context, p *Parser) ParseFn {
	const begin = 3

	line := p.Next()

	if !p.IsAtStart(line, []string{"RIB", "State:", "BGP", Skip}) {
		return t.errorf("did not have the RIB State as expected")
		return nil
	}

	s := ItemJoin(line, begin, -1)
	v, ok := toRIBState[s]
	if !ok {
		return t.errorf("did not have a valid RIB State, had: %q", s)
	}

	t.stats.RIBState = v
	return t.sendState
}

var toSendState = map[string]SendState {
	"in sync": RSSendSync, 
 	"not in sync": RSSendNotSync, 
 	"not advertising": RSSendNoAdvertise, 
}

// Send state: in sync
func (t *tableStats) sendState(ctx context.Context, p *Parser) ParseFn {
	const begin = 2

	line := p.Next()

	if !p.IsAtStart(line, []string{"Send", "state:", Skip}) {
		return t.errorf("did not have the Send state as expected")
		return nil
	}

	s := ItemJoin(line, begin, -1)
	v, ok := toSendState[s]
	if !ok {
		return t.errorf("did not have recognized Send state, had %s", s)
	}

	t.stats.SendState = v
	return t.active
}

// Active prefixes:              0
func (t *tableStats) active(ctx context.Context, p *Parser) ParseFn {
	i, err := t.intKeyVal([]string{"Active", "prefixes:", Skip}, p)
	if err != nil {
		return t.errorf(err.Error())
	}
	t.stats.ActivePrefixes = i
	return t.received
}

// Received prefixes:            0
func (t *tableStats) received(ctx context.Context, p *Parser) ParseFn {
	i, err := t.intKeyVal([]string{"Received", "prefixes:", Skip}, p)
	if err != nil {
		return t.errorf(err.Error())
	}
	t.stats.RecvPrefixes = i
	return t.accepted
}

// Accepted prefixes:            0
func (t *tableStats) accepted(ctx context.Context, p *Parser) ParseFn {
	i, err := t.intKeyVal([]string{"Accepted", "prefixes:", Skip}, p)
	if err != nil {
		return t.errorf(err.Error())
	}
	t.stats.AcceptPrefixes = i
	return t.supressed
}

// Suppressed due to damping:    0
func (t *tableStats) supressed(ctx context.Context, p *Parser) ParseFn {
	i, err := t.intKeyVal([]string{"Suppressed", "due", "to", "damping:", Skip}, p)
	if err != nil {
		return t.errorf(err.Error())
	}
	t.stats.SurpressedPrefixes = i
	return t.advertised
}

// Advertised prefixes:          0
func (t *tableStats) advertised(ctx context.Context, p *Parser) ParseFn {
	i, err := t.intKeyVal([]string{"Advertised", "prefixes:", Skip}, p)
	if err != nil {
		return t.errorf(err.Error())
	}
	t.stats.AdvertisedPrefixes = i
	return t.recordStats
}

func (t *tableStats) recordStats(ctx context.Context, p *Parser) ParseFn {
	if t.rec.InetStats == nil {
		t.rec.InetStats = map[int]*InetStats{}
	}
	t.rec.InetStats[t.stats.ID] = t.stats

	return t.peer.findTableStats
}

func (t *tableStats) intKeyVal(name []string, p *Parser) (int, error) {
	line := p.Next()
	if !p.IsAtStart(line, name) {
		return 0, fmt.Errorf("did not have %s as expected", strings.Join(name, " "))
	}

	item := line.Items[len(name)-1]
	v, err := item.ToInt()
	if err != nil {
		return 0,  fmt.Errorf("did not have %s value as a int, had %v", strings.Join(name, " "), item.Val)
	}
	return v, nil
} 

func ipPort(s string) (net.IP, int, error) {
	sp := strings.Split(s, `+`)
	if len(sp) != 2 {
		return nil, 0, fmt.Errorf("IP address and port could not be found with syntax <ip>+<port>: %s", s)
	}
	ip := net.ParseIP(sp[0])
	if ip == nil {
		return nil, 0, fmt.Errorf("IP address could not be parsed: %s", sp[0])
	}
	port, err := strconv.Atoi(sp[1])
	if err != nil {
		return nil, 0, fmt.Errorf("IP port could not be parsed from: %s", sp[1])
	}
	return ip, port, nil
}

























































