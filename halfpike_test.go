package halfpike

import (
	"testing"
	"github.com/kylelemons/godebug/pretty"
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
		{Type: ItemEOL, Val: "\n", lineNum: 1, raw: "\n\tPeer: 10.10.10.2+179 AS 22     Local: 10.10.10.1+65406 AS 17   \n"},
		{Type: ItemText, Val: "Type:"},
		{Type: ItemText, Val: "External"},
		{Type: ItemText, Val: "State:"},
		{Type: ItemText, Val: "Established"},
		{Type: ItemText, Val: "Flags:"},
		{Type: ItemText, Val: "<Sync>"},
		{Type: ItemEOL, Val: "\n", lineNum: 2, raw: "  Type: External    State: Established    Flags: <Sync>\n"},
		{Type: ItemEOF, lineNum: 3, raw: "\x01"},
	}	

	l := newLexer(str, untilSpace)
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
			Items: []Item{
				{Type: ItemText, Val: "Peer:"},
				{Type: ItemText, Val: "10.10.10.2+179"},
				{Type: ItemText, Val: "AS"},
				{Type: ItemInt, Val: "22"},
				{Type: ItemText, Val: "Local:"},
				{Type: ItemText, Val: "10.10.10.1+65406"},
				{Type: ItemText, Val: "AS"},
				{Type: ItemInt, Val: "17"},
				{Type: ItemEOL, Val: "\n", lineNum: 1, raw: "\n\tPeer: 10.10.10.2+179 AS 22     Local: 10.10.10.1+65406 AS 17   \n"},
			},
		},
		{
			Items: []Item{
				{Type: ItemText, Val: "Type:"},
				{Type: ItemText, Val: "External"},
				{Type: ItemText, Val: "State:"},
				{Type: ItemText, Val: "Established"},
				{Type: ItemText, Val: "Flags:"},
				{Type: ItemText, Val: "<Sync>"},
				{Type: ItemEOL, Val: "\n", lineNum: 2, raw: "  Type: External    State: Established    Flags: <Sync>\n"},
			},
		},
		{
			Items: []Item{
				{Type: ItemEOF, lineNum: 3, raw: "\x01"},
			},
		},
	}

	p, err := NewParser(str, nil)
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
	p, err := NewParser(str, nil)
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