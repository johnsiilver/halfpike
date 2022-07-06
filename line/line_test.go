package line

import (
	"log"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestLexer(t *testing.T) {
	tests := []struct {
		line string
		want []Item
	}{
		{
			line: "hello, how are $1doing Doing1$ 1.4 ab2c ac2  1024 2ac a3.2b b3.2 3.2a\n",
			want: []Item{
				{ItemText, "hello,"},
				{ItemSpace, " "},
				{ItemText, "how"},
				{ItemSpace, " "},
				{ItemText, "are"},
				{ItemSpace, " "},
				{ItemText, "$1doing"},
				{ItemSpace, " "},
				{ItemText, "Doing1$"},
				{ItemSpace, " "},
				{ItemFloat, "1.4"},
				{ItemSpace, " "},
				{ItemText, "ab2c"},
				{ItemSpace, " "},
				{ItemText, "ac2"},
				{ItemSpace, " "},
				{ItemSpace, " "},
				{ItemInt, "1024"},
				{ItemSpace, " "},
				{ItemText, "2ac"},
				{ItemSpace, " "},
				{ItemText, "a3.2b"},
				{ItemSpace, " "},
				{ItemText, "b3.2"},
				{ItemSpace, " "},
				{ItemText, "3.2a"},
				{ItemEOL, "\n"},
			},
		},
		{
			// Testing negative int, float and just a plain string with - before it.
			line: "-3.2 -1  \t-hello",
			want: []Item{
				{ItemFloat, "-3.2"},
				{ItemSpace, " "},
				{ItemInt, "-1"},
				{ItemSpace, " "},
				{ItemSpace, " "},
				{ItemSpace, "\t"},
				{ItemText, "-hello"},
				{ItemEOF, ""},
			},
		},
	}

	for _, test := range tests {
		lex := New(test.line)

		if diff := pretty.Compare(test.want, lex.items); diff != "" {
			t.Errorf("TestLexer: -want/+got:\n%s", diff)
		}
	}
}

func TestDecodeList(t *testing.T) {
	baseDL := DecodeList{LeftConstraint: "[", RightConstraint: "]", Separator: ",", EntryQuote: `"`}

	tests := []struct {
		desc string
		dl   DecodeList
		line string
		want []string
		err  bool
	}{
		{
			desc: "No left constraint",
			dl:   DecodeList{RightConstraint: "]", Separator: ",", EntryQuote: `"`},
			line: `[ "hello", "how", "are", "you" ]`,
			err:  true,
		},
		{
			desc: "No right constraint",
			dl:   DecodeList{LeftConstraint: "[", Separator: ",", EntryQuote: `"`},
			line: `[ "hello", "how", "are", "you" ]`,
			err:  true,
		},
		{
			desc: "No separator",
			dl:   DecodeList{LeftConstraint: "[", RightConstraint: "]", EntryQuote: `"`},
			line: `[ "hello", "how", "are", "you" ]`,
			err:  true,
		},
		{
			desc: "No entry quote",
			dl:   DecodeList{LeftConstraint: "[", RightConstraint: "]", Separator: ","},
			line: `[ "hello", "how", "are", "you" ]`,
			err:  true,
		},
		{
			dl:   baseDL,
			line: `[ "hello", "how", "are", "you" ]`,
			want: []string{"hello", "how", "are", "you"},
		},

		{
			dl:   baseDL,
			line: `["hello", "how", "are you",]`,
			want: []string{"hello", "how", "are you"},
		},

		{
			dl:   baseDL,
			line: `["hello", "how", "are you", ]`,
			want: []string{"hello", "how", "are you"},
		},

		{
			desc: "Using single quotes instead of my specified double quotes",
			dl:   baseDL,
			line: `[ 'hello', 'how', 'are', 'you' ]`,
			want: []string{"hello", "how", "are", "you"},
			err:  true,
		},
		{
			dl:   baseDL,
			line: `["hello","how","are you"]`,
			want: []string{"hello", "how", "are you"},
		},
		{
			dl:   baseDL,
			line: `[ "hello" , "how" , "are you" ]`,
			want: []string{"hello", "how", "are you"},
		},
	}

	for _, test := range tests {
		log.Println("\n\nNew test!!!")

		l := New(test.line)

		got, err := test.dl.Decode(l)
		switch {
		case err == nil && test.err:
			if test.desc == "" {
				t.Errorf("Test(%q): got err == nil, want err != nil", test.line)
			} else {
				t.Errorf("Test(%s): got err == nil, want err != nil", test.desc)
			}
			continue
		case err != nil && !test.err:
			if test.desc == "" {
				t.Errorf("Test(%q): got err == %s, want err != nil", test.line, err)
			} else {
				t.Errorf("Test(%s): got err == %s, want err != nil", test.desc, err)
			}
			continue
		case err != nil:
			continue
		}

		if diff := pretty.Compare(test.want, got); diff != "" {
			t.Errorf("Test(%q): -want/+got:\n%s", test.line, diff)
		}
	}
}
