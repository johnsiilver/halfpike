package halfpike

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kylelemons/godebug/pretty"
)

var showIntBrief = `
Doesn't matter what comes before
what we are looking for
Physical interface: ge-3/0/2, Enabled, Physical link is Up
  Link-level type: 52, MTU: 1522, Speed: 1000mbps, Loopback: Disabled,
  This is just some trash
Physical interface: ge-3/0/3, Enabled, Physical link is Up
  Link-level type: ppp, MTU: 1522, Speed: 1000mbps, Loopback: Disabled,
  This doesn't matter either
`

func Example_short() {
	inters := Interfaces{}

	// Parses our content in showBGPNeighbor and begins parsing with states.FindPeer
	// which is a ParseFn.
	if err := Parse(context.Background(), showIntBrief, &inters); err != nil {
		panic(err)
	}

	// Because we pass in a slice, we have to do a reassign to get the changed value.
	fmt.Println(pretty.Sprint(inters.Interfaces))

	// Leaving off the output: line, because getting the output to line up after vsc reformats
	// is just horrible.
	/*
	   [{VendorDesc: "ge-3/0/2",
	     Blade:      3,
	     Pic:        0,
	     Port:       2,
	     State:      1,
	     Status:     1,
	     LinkLevel:  1,
	     MTU:        1522,
	     Speed:      1000000000},
	    {VendorDesc: "ge-3/0/3",
	     Blade:      3,
	     Pic:        0,
	     Port:       3,
	     State:      1,
	     Status:     1,
	     LinkLevel:  2,
	     MTU:        1522,
	     Speed:      1000000000}]
	*/
}

type LinkLevel int8

const (
	LLUnknown  LinkLevel = 0
	LL52       LinkLevel = 1
	LLPPP      LinkLevel = 2
	LLEthernet LinkLevel = 3
)

type InterState int8

const (
	IStateUnknown  InterState = 0
	IStateEnabled  InterState = 1
	IStateDisabled InterState = 2
)

type InterStatus int8

const (
	IStatUnknown InterStatus = 0
	IStatUp      InterStatus = 1
	IStatDown    InterStatus = 2
)

// Interfaces is a collection of Interface information for a device.
type Interfaces struct {
	Interfaces []*Interface

	parser *Parser
}

func (i *Interfaces) Validate() error {
	for _, v := range i.Interfaces {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (i *Interfaces) errorf(s string, a ...interface{}) ParseFn {
	if len(i.Interfaces) > 0 {
		v := i.current().VendorDesc
		if v != "" {
			return i.parser.Errorf("interface(%s): %s", v, fmt.Sprintf(s, a...))
		}
	}
	return i.parser.Errorf(s, a...)
}

func (i *Interfaces) Start(ctx context.Context, p *Parser) ParseFn {
	return i.findInterface
}

var phyStart = []string{"Physical", "interface:", Skip, Skip, "Physical", "link", "is", Skip}

// Physical interface: ge-3/0/2, Enabled, Physical link is Up
func (i *Interfaces) findInterface(ctx context.Context, p *Parser) ParseFn {
	if i.parser == nil {
		i.parser = p
	}

	// The Skip here says that we need to have an item here, but we don't care what it is.
	// This way we can deal with dynamic values and ensure we
	// have the minimum values we need.
	// p.FindItemsRegexStart() can be used if you require more
	// complex matching of static values.
	_, err := p.FindStart(phyStart)
	if err != nil {
		if len(i.Interfaces) == 0 {
			return i.errorf("could not find a physical interface in the output")
		}
		return nil
	}
	// Create our new entry.
	inter := &Interface{}
	inter.init()
	i.Interfaces = append(i.Interfaces, inter)

	p.Backup() // I like to start all ParseFn with either Find...() or p.Next() for consistency.
	return i.phyInter
}

var toInterState = map[string]InterState{
	"Enabled,":  IStateEnabled,
	"Disabled,": IStateDisabled,
}

var toStatus = map[string]InterStatus{
	"Up":   IStatUp,
	"Down": IStatDown,
}

// Physical interface: ge-3/0/2, Enabled, Physical link is Up
func (i *Interfaces) phyInter(ctx context.Context, p *Parser) ParseFn {
	// These are indexes within the line where our values are.
	const (
		name        = 2
		stateIndex  = 3
		statusIndex = 7
	)
	line := p.Next() // fetches the next line of ouput.

	i.current().VendorDesc = line.Items[name].Val[:len(line.Items[name].Val)-1] // this will be ge-3/0/2 in the example above
	if err := i.interNameSplit(line.Items[name].Val); err != nil {
		return i.errorf("error parsing the name into blade/pic/port: %s", err)
	}

	state, ok := toInterState[line.Items[stateIndex].Val]
	if !ok {
		return i.errorf("error parsing the interface state, got %s is not a known state", line.Items[stateIndex].Val)
	}
	i.current().State = state

	status, ok := toStatus[line.Items[statusIndex].Val]
	if !ok {
		return i.errorf("error parsing the interface status, got %s which is not a known status", line.Items[statusIndex].Val)
	}
	i.current().Status = status
	return i.findLinkLevel
}

var toLinkLevel = map[string]LinkLevel{
	"52,":       LL52,
	"ppp,":      LLPPP,
	"ethernet,": LLEthernet,
}

// Link-level type: 52, MTU: 1522, Speed: 1000mbps, Loopback: Disabled,
func (i *Interfaces) findLinkLevel(ctx context.Context, p *Parser) ParseFn {
	const (
		llTypeIndex = 2
		mtuIndex    = 4
		speedIndex  = 6
	)

	line, until, err := p.FindUntil([]string{"Link-level", "type:", Skip, "MTU:", Skip, "Speed:", Skip}, phyStart)
	if err != nil {
		return i.errorf("did not find Link-level before end of file reached")
	}
	if until {
		return i.errorf("did not find Link-level before finding the next interface")
	}

	ll, ok := toLinkLevel[line.Items[llTypeIndex].Val]
	if !ok {
		return i.errorf("unknown link level type: %s", line.Items[llTypeIndex].Val)
	}
	i.current().LinkLevel = ll

	mtu, err := strconv.Atoi(strings.Split(line.Items[mtuIndex].Val, ",")[0])
	if err != nil {
		return i.errorf("mtu did not seem to be a valid integer: %s", line.Items[mtuIndex].Val)
	}
	i.current().MTU = mtu

	if err := i.speedSplit(line.Items[speedIndex].Val); err != nil {
		return i.errorf("problem interpreting the interface speed: %s", err)
	}

	return i.findInterface
}

// ge-3/0/2
var interNameRE = regexp.MustCompile(`(?P<inttype>ge)-(?P<blade>\d+)/(?P<pic>\d+)/(?P<port>\d+),`)

func (i *Interfaces) interNameSplit(s string) error {
	matches, err := Match(interNameRE, s)
	if err != nil {
		return fmt.Errorf("error disecting the interface name(%s): %s", s, err)
	}

	for k, v := range matches {
		if k == "inttype" {
			continue
		}
		in, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("could not convert value for %s(%s) to an integer", k, v)
		}
		switch k {
		case "blade":
			i.current().Blade = in
		case "pic":
			i.current().Pic = in
		case "port":
			i.current().Port = in
		}
	}
	return nil
}

var speedRE = regexp.MustCompile(`(?P<bits>\d+)(?P<desc>(kbps|mbps|gbps))`)
var bitsMultiplier = map[string]int{
	"kbps": 1000,
	"mbps": 1000 * 1000,
	"gbps": 1000 * 1000 * 1000,
}

func (i *Interfaces) speedSplit(s string) error {
	matches, err := Match(speedRE, s)
	if err != nil {
		return fmt.Errorf("error disecting the interfacd speed(%s): %s", s, err)
	}

	multi, ok := bitsMultiplier[matches["desc"]]
	if !ok {
		return fmt.Errorf("could not decipher the interface speed measurement: %s", matches["desc"])
	}

	bits, err := strconv.Atoi(matches["bits"])
	if err != nil {
		return fmt.Errorf("interface speed does not seem to be a integer: %s", matches["bits"])
	}
	i.current().Speed = bits * multi
	return nil
}

func (i *Interfaces) current() *Interface {
	if len(i.Interfaces) == 0 {
		return nil
	}
	return i.Interfaces[len(i.Interfaces)-1]
}

// Interface is a brief decription of a network interface.
type Interface struct {
	// VendorDesc is the name a vendor gives the interface, like ge-10/2/1.
	VendorDesc string
	// Blade is the blade in the routing chassis.
	Blade int
	// Pic is the pic position on the blade.
	Pic int
	// Port is the port in the pic.
	Port int
	// State is the interface's current state.
	State InterState
	// Status is the interface's current status.
	Status InterStatus
	// LinkLevel is the type of encapsulation used on the link.
	LinkLevel LinkLevel
	// MTU is the maximum amount of bytes that can be sent on the frame.
	MTU int
	// Speed is the interface's speed in bits per second.
	Speed int

	initCalled bool
}

// init initializes Interface.
func (i *Interface) init() {
	i.Blade = -1
	i.Pic = -1
	i.Port = -1
	i.MTU = -1
	i.Speed = -1
	i.initCalled = true
}

// Validate implements halfpike.Validator.
func (i *Interface) Validate() error {
	if !i.initCalled {
		return fmt.Errorf("an Interface did not have init() called before storing data")
	}

	if i.VendorDesc == "" {
		return fmt.Errorf("an Interface did not have VendorDesc assigned")
	}

	switch -1 {
	case i.Blade:
		return fmt.Errorf("Interface(%s): Blade was not set", i.VendorDesc)
	case i.Pic:
		return fmt.Errorf("Interface(%s): Pic was not set", i.VendorDesc)
	case i.Port:
		return fmt.Errorf("Interface(%s): Port was not set", i.VendorDesc)
	case i.MTU:
		return fmt.Errorf("Interface(%s): MTU was not set", i.VendorDesc)
	case i.Speed:
		return fmt.Errorf("Interface(%s): Speed was not set", i.VendorDesc)
	}

	switch {
	case i.State == IStateUnknown:
		return fmt.Errorf("Interface(%s): State was not set", i.VendorDesc)
	case i.Status == IStatUnknown:
		return fmt.Errorf("Interface(%s): Status was not set", i.VendorDesc)
	case i.LinkLevel == LLUnknown:
		return fmt.Errorf("Interface(%s): LinkLevel was not set", i.VendorDesc)
	}

	return nil
}
