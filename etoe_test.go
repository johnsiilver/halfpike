package halfpike

import (
	"context"
	//"fmt"
	"net"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
)

func TestEndToEnd(t *testing.T) {
	neighbors := BGPNeighbors{}
	p, err := NewParser(showBGPNeighbor, neighbors)
	if err != nil {
		panic(err)
	}

	states := &PeerParser{}

	if err := Parse(context.Background(), p, states.FindPeer); err != nil {
		panic(err)
	}

	neighbors = p.Validator.(BGPNeighbors)

	want := BGPNeighbors{
		{
			PeerIP:     net.ParseIP("10.10.10.2"),
			PeerPort:   179,
			PeerAS:     22,
			LocalIP:    net.ParseIP("10.10.10.1"),
			LocalPort:  65406,
			LocalAS:    22,
			Type:       1,
			State:      3,
			LastState:  5,
			HoldTime:   90 * time.Second,
			Preference: 170,
			PeerID:     net.ParseIP("10.10.10.2"),
			LocalID:    net.ParseIP("10.10.10.1"),
			InetStats: map[int]*InetStats{
				0: &InetStats{
					ID:                 0,
					Bit:                10000,
					RIBState:           2,
					SendState:          1,
					ActivePrefixes:     0,
					RecvPrefixes:       0,
					AcceptPrefixes:     0,
					SurpressedPrefixes: 2,
					AdvertisedPrefixes: 0,
				},
			},
			initCalled: true,
		},
		{
			PeerIP:     net.ParseIP("10.10.10.6"),
			PeerPort:   54781,
			PeerAS:     22,
			LocalIP:    net.ParseIP("10.10.10.5"),
			LocalPort:  179,
			LocalAS:    22,
			Type:       1,
			State:      3,
			LastState:  5,
			HoldTime:   90 * time.Second,
			Preference: 170,
			PeerID:     net.ParseIP("10.10.10.6"),
			LocalID:    net.ParseIP("10.10.10.1"),
			InetStats: map[int]*InetStats{
				0: &InetStats{
					ID:                 0,
					Bit:                10000,
					RIBState:           2,
					SendState:          1,
					ActivePrefixes:     0,
					RecvPrefixes:       0,
					AcceptPrefixes:     0,
					SurpressedPrefixes: 0,
					AdvertisedPrefixes: 0,
				},
			},
			initCalled: true,
		},
	}

	if diff := pretty.Compare(want, neighbors); diff != "" {
		t.Errorf("TestEndToEnd: -want/+got:\n%s", diff)
	}
}
