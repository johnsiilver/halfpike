package line

import (
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
