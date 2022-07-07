package halfpike

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	//"fmt"
)

const str = `
	Peer: 10.10.10.2+179 AS 22     Local: 10.10.10.1+65406 AS 17   
  Type: External    State: Established    Flags: <Sync>
`

func TestLexer(t *testing.T) {
	want := []Item{
		{Type: ItemText, Val: "Peer:"},
		{Type: ItemText, Val: "10.10.10.2+179"},
		{Type: ItemText, Val: "AS"},
		{Type: ItemInt, Val: "22"},
		{Type: ItemText, Val: "Local:"},
		{Type: ItemText, Val: "10.10.10.1+65406"},
		{Type: ItemText, Val: "AS"},
		{Type: ItemInt, Val: "17"},
		{Type: ItemEOL, Val: "\n", lineNum: 1, raw: "\tPeer: 10.10.10.2+179 AS 22     Local: 10.10.10.1+65406 AS 17   \n"},
		{Type: ItemText, Val: "Type:"},
		{Type: ItemText, Val: "External"},
		{Type: ItemText, Val: "State:"},
		{Type: ItemText, Val: "Established"},
		{Type: ItemText, Val: "Flags:"},
		{Type: ItemText, Val: "<Sync>"},
		{Type: ItemEOL, Val: "\n", lineNum: 2, raw: "  Type: External    State: Established    Flags: <Sync>\n"},
		{Type: ItemEOF, lineNum: 3, raw: "\x01"},
	}

	l := newLexer(context.Background(), str, untilEOF)
	go l.run()

	got := []Item{}
	for item := range l.items {
		got = append(got, item)
	}

	if diff := pretty.Compare(want, got); diff != "" {
		t.Errorf("TestLexer: -want/+got:\n%s", diff)
	}
}

func TestNext(t *testing.T) {
	want := []Line{
		{
			LineNum: 1,
			Raw:     "\tPeer: 10.10.10.2+179 AS 22     Local: 10.10.10.1+65406 AS 17   \n",
			Items: []Item{
				{Type: ItemText, Val: "Peer:"},
				{Type: ItemText, Val: "10.10.10.2+179"},
				{Type: ItemText, Val: "AS"},
				{Type: ItemInt, Val: "22"},
				{Type: ItemText, Val: "Local:"},
				{Type: ItemText, Val: "10.10.10.1+65406"},
				{Type: ItemText, Val: "AS"},
				{Type: ItemInt, Val: "17"},
				{Type: ItemEOL, Val: "\n"},
			},
		},
		{
			LineNum: 2,
			Raw:     "  Type: External    State: Established    Flags: <Sync>\n",
			Items: []Item{
				{Type: ItemText, Val: "Type:"},
				{Type: ItemText, Val: "External"},
				{Type: ItemText, Val: "State:"},
				{Type: ItemText, Val: "Established"},
				{Type: ItemText, Val: "Flags:"},
				{Type: ItemText, Val: "<Sync>"},
				{Type: ItemEOL, Val: "\n"},
			},
		},
		{
			LineNum: 3,
			Raw:     "\x01",
			Items: []Item{
				{Type: ItemEOF},
			},
		},
	}

	p, err := newParser(str)
	if err != nil {
		panic(err)
	}
	go p.lex.run()

	got := []Line{}
	for line := p.Next(); true; line = p.Next() {
		got = append(got, line)
		if p.EOF(line) {
			break
		}
	}

	if diff := pretty.Compare(want, got); diff != "" {
		t.Errorf("TestNext: -want/+got:\n%s", diff)
	}
}

func TestBackup(t *testing.T) {
	p, err := newParser(str)
	if err != nil {
		panic(err)
	}
	go p.lex.run()

	lines := []Line{}
	for line := p.Next(); true; line = p.Next() {
		lines = append(lines, line)
		if p.EOF(line) {
			break
		}
	}

	reverse := make([]Line, len(lines))
	for i := 0; i < len(lines); i++ {
		reverse[len(lines)-1-i] = p.Backup()
	}

	if diff := pretty.Compare(lines, reverse); diff != "" {
		t.Errorf("TestBackup: -want/+got:\n%s", diff)
	}
}

func TestFindStart(t *testing.T) {
	tests := []struct {
		desc string
		find []string
		want []Item
		err  bool

		reset bool
	}{
		{
			desc: "Find entry toward the middle",
			find: []string{"Local", "Interface:", "ge-1/2/0.0"},
			want: []Item{
				{Type: 2, Val: "Local"},
				{Type: 2, Val: "Interface:"},
				{Type: 2, Val: "ge-1/2/0.0"},
				{Type: 5, Val: "\n"},
			},
			err: false,
		},
		{
			desc: "Use Any to find the next entry",
			find: []string{"Send", "state:", Skip, "sync"},
			want: []Item{
				{Type: 2, Val: "Send"},
				{Type: 2, Val: "state:"},
				{Type: 2, Val: "in"},
				{Type: 2, Val: "sync"},
				{Type: 5, Val: "\n"},
			},
			err: false,
		},
		{
			desc: "Can't find the entry (we have already passed it in the input)",
			find: []string{"Local", "Interface:", "ge-1/2/0.0"},
			want: []Item{
				{Type: 2, Val: "Send"},
				{Type: 2, Val: "state:"},
				{Type: 2, Val: "in"},
				{Type: 2, Val: "sync"},
				{Type: 5, Val: "\n"},
			},
			err: true,
		},
		{
			desc:  "Too many search parameters lead to no match",
			find:  []string{"Send", "state:", Skip, "sync", Skip, Skip}, // You have to also include a Skip for EOL or EOF
			err:   true,
			reset: true,
		},
	}

	var p *Parser
	for _, test := range tests {
		if test.reset || p == nil {
			var err error
			p, err = newParser(showBGPNeighbor)
			if err != nil {
				panic(err)
			}
			go p.lex.run()
		}
		got, err := p.FindStart(test.find)
		switch {
		case test.err && err == nil:
			t.Fatalf("TestFindStart(%s): got err == nil, want err != nil", test.desc)
		case !test.err && err != nil:
			t.Fatalf("TestFindStart(%s): got err == %s, want err == nil", test.desc, err)
		case err != nil:
			continue
		}
		if diff := pretty.Compare(test.want, got.Items); diff != "" {
			t.Fatalf("TestFindStart(%s): -want/+got:\n%s", test.find, diff)
		}
	}
}

func TestFindUntil(t *testing.T) {
	p, err := newParser(showBGPNeighbor)
	if err != nil {
		panic(err)
	}
	go p.lex.run()

	tests := []struct {
		desc      string
		startFn   bool
		untilFn   bool
		find      []string
		until     []string
		want      []Item
		wantUntil bool
		err       bool
	}{
		{
			desc:    "Find record start",
			find:    peerRecStart,
			startFn: true,
			want: []Item{
				{Type: 2, Val: "Peer:"},
				{Type: 2, Val: "10.10.10.2+179"},
				{Type: 2, Val: "AS"},
				{Type: 3, Val: "22"},
				{Type: 2, Val: "Local:"},
				{Type: 2, Val: "10.10.10.1+65406"},
				{Type: 2, Val: "AS"},
				{Type: 3, Val: "17"},
				{Type: 5, Val: "\n"},
			},
			err: false,
		},
		{
			desc:    "Find sub record",
			untilFn: true,
			find:    []string{"Send", "state:", Skip, "sync"},
			until:   peerRecStart,
			want: []Item{
				{Type: 2, Val: "Send"},
				{Type: 2, Val: "state:"},
				{Type: 2, Val: "in"},
				{Type: 2, Val: "sync"},
				{Type: 5, Val: "\n"},
			},
			err: false,
		},
		{
			desc:      "Attempt to find sub record, but instead find next record",
			untilFn:   true,
			find:      []string{"Send", "state:", Skip, "sync"},
			until:     peerRecStart,
			wantUntil: true,
			err:       false,
		},
		{
			desc:    "Find next record start",
			find:    peerRecStart,
			startFn: true,
			want: []Item{
				{Type: 2, Val: "Peer:"},
				{Type: 2, Val: "10.10.10.6+54781"},
				{Type: 2, Val: "AS"},
				{Type: 3, Val: "22"},
				{Type: 2, Val: "Local:"},
				{Type: 2, Val: "10.10.10.5+179"},
				{Type: 2, Val: "AS"},
				{Type: 3, Val: "17"},
				{Type: 5, Val: "\n"},
			},
			err: false,
		},
		{
			desc:    "Find sub record",
			untilFn: true,
			find:    []string{"Send", "state:", Skip, "sync"},
			until:   peerRecStart,
			want: []Item{
				{Type: 2, Val: "Send"},
				{Type: 2, Val: "state:"},
				{Type: 2, Val: "in"},
				{Type: 2, Val: "sync"},
				{Type: 5, Val: "\n"},
			},
			err: false,
		},
		{
			desc:    "Attempt to find sub record, but instead find EOF",
			untilFn: true,
			find:    []string{"Send", "state:", Skip, "sync"},
			until:   peerRecStart,
			err:     true,
		},
	}

	for _, test := range tests {
		if test.startFn {
			got, err := p.FindStart(test.find)
			switch {
			case test.err && err == nil:
				t.Fatalf("TestFindUntil(%s): got err == nil, want err != nil", test.desc)
			case !test.err && err != nil:
				t.Fatalf("TestFindUntil(%s): got err == %s, want err == nil", test.desc, err)
			case err != nil:
				continue
			}
			if diff := pretty.Compare(test.want, got.Items); diff != "" {
				t.Fatalf("TestFindUntil(%s): -want/+got:\n%s", test.find, diff)
			}
		} else {
			got, until, err := p.FindUntil(test.find, test.until)
			switch {
			case test.err && err == nil:
				t.Fatalf("TestFindUntil(%s): got err == nil, want err != nil", test.desc)
			case !test.err && err != nil:
				t.Fatalf("TestFindUntil(%s): got err == %s, want err == nil", test.desc, err)
			case err != nil:
				continue
			case until != test.wantUntil:
				t.Fatalf("TestFindUntil(%s): got until == %v, want until == %v", test.desc, until, !until)
			}

			if diff := pretty.Compare(test.want, got.Items); diff != "" {
				t.Fatalf("TestFindUntil(%s): -want/+got:\n%s", test.find, diff)
			}
		}
	}
}

type startWithCarriageObj struct{}

func (s *startWithCarriageObj) Start(ctx context.Context, p *Parser) ParseFn {
	for {
		line := p.Next()
		if strings.HasPrefix(line.Raw, "\n") {
			log.Printf("raw: %q", line.Raw)
			return p.Errorf("[LineNum %d]: line.Raw begins with \\n", line.LineNum)
		}
		if p.EOF(line) {
			return nil
		}
	}
}

func (s *startWithCarriageObj) Validate() error {
	return nil
}

func TestRegressionRawStartsWithCarriageReturn(t *testing.T) {
	f, err := os.ReadFile("./testing/testfile.claw")
	if err != nil {
		panic(err)
	}

	obj := &startWithCarriageObj{}

	if err := Parse(context.Background(), string(f), obj); err != nil {
		t.Fatalf("TestRegressionRawStartsWithCarriageReturn: got err == %s", err)
	}
}
