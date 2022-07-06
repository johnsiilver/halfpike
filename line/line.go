// Package line provides a secondary lexer when you need a more grandular lexer for a single line.
// Unlike the normal halfpike lexer, we do not ignore spaces here and it let's you ask deeper questions
// on an item.
package line

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

//go:generate stringer -type=ItemType

// ItemType describes the type of item being emitted by the Lexer. The numeric values may change
// between versions, so they cannot be recorded to disk and relied upon.
type ItemType int

const (
	// ItemUnknown indicates that the Item is an unknown. This should only happen on
	// a Item that is the zero type.
	Unknown ItemType = iota
	// ItemEOF indicates that the end of input is reached. No further tokens will be sent.
	ItemEOF
	// ItemText indicates that it is a block of text separated by some type of space (including tabs).
	// This may contain numbers, but only if it is mixed with letters.
	ItemText
	// ItemInt indicates that an integer was found.
	ItemInt
	// ItemFloat indicates that a float was found.
	ItemFloat
	// ItemEOL indicates the end of a line was reached.
	ItemEOL
	// ItemSpace indicates a space character as recognized by unicode.IsSpace().
	// This is only emitted by the line.Lexer, never by HalfPike.
	ItemSpace
)

// Item represents a token created by the Lexer.
type Item struct {
	// Type is the type of item that is stored in .Val.
	Type ItemType
	// Val is the value of the item that was in the text output.
	Val string
}

// IsZero indicates the Item is the zero value.
func (i Item) IsZero() bool {
	return reflect.ValueOf(i).IsZero()
}

// HasPrefix returns true if the Item starts with prefix.
func (i Item) HasPrefix(prefix string) bool {
	if i.Type != ItemText {
		return false
	}
	return strings.HasPrefix(i.Val, prefix)
}

// HasSuffix returns true if the Item ends with suffix.
func (i Item) HasSuffix(suffix string) bool {
	if i.Type != ItemText {
		return false
	}
	return strings.HasSuffix(i.Val, suffix)
}

// Capitalized indicates the first letter is capitalized.
func (i Item) Capitalized() bool {
	if i.Type != ItemText {
		return false
	}
	if strings.ToUpper(i.Val) == i.Val {
		return true
	}
	return false
}

// StartsWithLetter indicates if the text begins with a letter.
func (i Item) StartsWithLetter() bool {
	if i.Type != ItemText {
		return false
	}
	if len(i.Val) == 0 {
		return false
	}

	runes := []rune(i.Val)
	return unicode.IsLetter(runes[0])
}

// OnlyLetters returns true if the Item is made up of only letters.
func (i Item) OnlyLetters() bool {
	if i.Type != ItemText {
		return false
	}
	if len(i.Val) == 0 {
		return false
	}
	for _, r := range i.Val {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// OnlyLettersAndNumbers returns true if only letters and numbers are present.
func (i Item) OnlyLettersAndNumbers() bool {
	if i.Type != ItemText {
		return false
	}
	if len(i.Val) == 0 {
		return false
	}
	for _, r := range i.Val {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}

// OnlyHas returns if the Item only has a combination of letters(if true), numbers(if true) and
// other runes you define (like '_', '-', ':', ',').
func (i Item) OnlyHas(letters, numbers bool, others ...rune) bool {
	if len(i.Val) == 0 {
		return false
	}
	for _, r := range i.Val {
		if letters && unicode.IsLetter(r) {
			continue
		}
		if numbers && unicode.IsNumber(r) {
			continue
		}
		cont := false
		for _, other := range others {
			if r == other {
				cont = true
				break
			}
		}
		if cont {
			continue
		}
		return false
	}
	return true
}

func (i Item) ContainsNumbers() bool {
	switch i.Type {
	case ItemText, ItemInt, ItemFloat:
		return true
	}
	for _, r := range i.Val {
		if unicode.IsNumber(r) {
			return true
		}
	}
	return false
}

// ASCIIOnly returns true if all the characters are ASCII characters.
func (i Item) ASCIIOnly() bool {
	for _, r := range i.Val {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// ToInt returns the value as an int type. If the Item.Type is not ItemInt, this will panic.
func (i Item) ToInt() (int, error) {
	if i.Type != ItemInt {
		return 0, fmt.Errorf("cannot convert %q to an int type", i.Val)
	}
	n, err := strconv.Atoi(i.Val)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// ToFloat returns the value as a float64 type. if the Item.Type is not itemFloat, this will panic.
func (i Item) ToFloat() (float64, error) {
	if i.Type != ItemFloat {
		return 0.0, fmt.Errorf("cannot convert %q to float64 type", i.Val)
	}

	f, err := strconv.ParseFloat(i.Val, 64)
	if err != nil {
		return 0.0, err
	}
	return f, nil
}

func ItemJoin(items ...Item) string {
	b := strings.Builder{}
	for _, i := range items {
		b.WriteString(i.Val)
	}
	return b.String()
}

// Lexer creates a new lexer for the line that can emit tokens.
type Lexer struct {
	index int
	items []Item
}

// New creates a new Lexer and lexes the line into Item(s) ready to parse.
func New(line string) *Lexer {
	items := []Item{}

	buff := strings.Builder{}
	var isNumber bool
	var isFloat bool
	for _, r := range line {
		switch {
		case unicode.IsSpace(r):
			if buff.Len() > 0 {
				switch {
				case isNumber && isFloat:
					items = append(items, Item{ItemFloat, buff.String()})
					buff.Reset()
				case isNumber:
					items = append(items, Item{ItemInt, buff.String()})
					buff.Reset()
				default:
					items = append(items, Item{ItemText, buff.String()})
					buff.Reset()
				}
				isNumber = false
				isFloat = false
			}
			if r == '\n' {
				items = append(items, Item{ItemEOL, string(r)})
			} else {
				items = append(items, Item{ItemSpace, string(r)})
			}
		case unicode.IsNumber(r):
			if buff.Len() == 0 {
				isNumber = true
			}
			buff.WriteRune(r)
		case r == '.':
			if isNumber {
				isFloat = true
			}
			buff.WriteRune(r)
		default:
			isNumber = false
			isFloat = false

			if len(items) == 0 || items[len(items)-1].Type == ItemSpace && buff.Len() == 0 {
				if r == '-' {
					isNumber = true
				}
			}
			buff.WriteRune(r)
		}
	}

	if buff.Len() > 0 {
		switch {
		case isNumber && isFloat:
			items = append(items, Item{ItemFloat, buff.String()})
			buff.Reset()
		case isNumber:
			items = append(items, Item{ItemInt, buff.String()})
			buff.Reset()
		default:
			items = append(items, Item{ItemText, buff.String()})
			buff.Reset()
		}
	}

	if items[len(items)-1].Type != ItemEOL {
		items = append(items, Item{Type: ItemEOF})
	}

	return &Lexer{
		items: items,
	}
}

// Next gets the next item from the line. Once you receive an EOL or EOF, any subsequent
// Next() calls will return that same Item.
func (l *Lexer) Next() Item {
	if l.index < len(l.items) {
		item := l.items[l.index]
		if l.items[l.index].Type != ItemEOL {
			l.index++
		}
		return item
	}
	return l.items[len(l.items)-1]
}

// Backup goes back 1 Item.
func (l *Lexer) Backup() {
	if l.index-1 >= 0 {
		l.index--
	}
}

// Peek returns the next Item, unless you have already reached EOF or EOL. In that case it
// simply returns that value.
func (l *Lexer) Peek() Item {
	n := l.Next()
	l.Backup()

	return n
}

// SetIndex will set the internal index number to the value of the item that will be read
// the next time .Next() is called.
func (l *Lexer) SetIndex(i int) {
	if i < 0 {
		log.Fatalf("cannot set negative index")
	}
	if i >= len(l.items) {
		log.Fatalf("cannot set index outside slice length")
	}
	l.index = i
}

// Len is the number of items in the Lexer.
func (l *Lexer) Len() int {
	return len(l.items)
}

// Index is the current index we are on. It is important to remember that this means we haven't
// read the value at the current index.
func (l *Lexer) Index() int {
	return l.index
}

// Range will emit items from the current Index until the end. You must either cancel the Context
// or read the entire channel, otherwise you will have a goroutine leak.
func (l *Lexer) Range(ctx context.Context) chan Item {
	ch := make(chan Item)

	go func() {
		defer close(ch)

		for {
			i := l.Next()
			select {
			case <-ctx.Done():
				l.Backup()
				return
			case ch <- i:
				switch i.Type {
				case ItemEOL, ItemEOF:
					return
				}
			}
		}
	}()

	return ch
}

// SkipAllSpaces skips over any space character (\n, ' ', \t, ...).
func SkipAllSpaces(l *Lexer) {
	for {
		r := l.Next()
		switch r.Type {
		case ItemSpace:
			continue
		}
		break
	}
	l.Backup()
}

// Skip skips over any set of strings.
func Skip(strs []string, l *Lexer) {
	for {
		cont := false
		i := l.Next()
		for _, m := range strs {
			if i.Val == m {
				cont = true
				break
			}
		}
		if !cont {
			break
		}
	}
	l.Backup()
}

// DecodeList can be used to decode a list of items, such as  ["hello", "I", "must", "be", "going"].
// It can handle escapes for your entry quotes, so if you use "entry", you can do "entr\"y".
// It will handle spaces around your entry separators, so ["entry", "entry"] and ["entry" , "entry"]
// are both tolerated, as well as spaces like [ "entry" ]. You can even do ["entry",].
// The big limitations are that I only support single character constraints and separators. So
// you can't do {{"entry"}} or [''entry''] types of lists.
type DecodeList struct {
	// LeftConstraint is the rune that indicates the beginning of the list, usually {, [, { .
	LeftConstraint string
	// RightConstraint is the rune that indicates the end of the list, usually }, ], } .
	RightConstraint string

	// Separator is the string that indicates separation between items. The separator
	// cannot be a space character.
	Separator string

	// EntryQuote indicates if the items are in quotes and if so, what the quote type is.
	// Must be either ' or ""
	EntryQuote string

	// EscapeCharater indicates the escape charater for quotes, if there is one.
	EscapeCharacter string

	results []string
}

func (d *DecodeList) setup() error {
	d.results = nil

	if d.LeftConstraint == "" {
		return errors.New(".LeftConstraint cannot be empty")
	}
	if d.RightConstraint == "" {
		return errors.New(".RightConstraint cannot be empty")
	}
	if len(d.Separator) != 1 {
		return errors.New(".Separator cannot be empty or more than 1 character")
	}
	if unicode.IsSpace([]rune(d.Separator)[0]) {
		return errors.New(".Separator cannot be a space character")
	}
	if d.EntryQuote == "" {
		return errors.New(`.EntryQuote must be " or '`)
	}
	if d.EntryQuote != `"` && d.EntryQuote != `'` {
		return errors.New(`.EntryQuote must be " or '`)
	}
	return nil
}

// Decode decodes a list as defined in DecodeList's attributes into items. When complete,
// the lexer will be right past the closing RightConstraint of the list, unless there was an error.
func (d *DecodeList) Decode(l *Lexer) (items []string, err error) {
	if err := d.setup(); err != nil {
		return nil, err
	}

	foundLeftConstraint := false
	foundRightConstraint := false
	inQuote := false
	lc := []rune(d.LeftConstraint)[0]
	rc := []rune(d.RightConstraint)[0]
	sep := []rune(d.Separator)[0]
	q := []rune(d.EntryQuote)[0]
	var escape rune
	if d.EscapeCharacter != "" {
		escape = []rune(d.EscapeCharacter)[0]
	}

	var entry strings.Builder
	read := 0
	var last rune
	var lastNonSpace rune

	for {
		if foundRightConstraint {
			return d.results, nil
		}
		i := l.Next()
		switch i.Type {
		case ItemEOF, ItemEOL:
			return nil, fmt.Errorf("list did not have ending character")
		}

		reader := strings.NewReader(i.Val)

		for reader.Len() > 0 {
			err := func() error {
				r, _, _ := reader.ReadRune()
				defer func() {
					last = r
					if !unicode.IsSpace(r) {
						lastNonSpace = r
					}
				}()

				read++
				if read == 2 {
					if !foundLeftConstraint {
						return fmt.Errorf("list should receive first character %q, but got '%c'", d.LeftConstraint, r)
					}
				}
				switch {
				case r == lc:
					if foundLeftConstraint && !inQuote {
						return fmt.Errorf("found the two %q in the list", d.LeftConstraint)
					}
					if inQuote {
						entry.WriteRune(r)
						return nil
					}
					foundLeftConstraint = true
					return nil
				case unicode.IsSpace(r):
					if inQuote {
						entry.WriteRune(r)
						return nil
					}
					return nil
				case r == q:
					if escape != 0 && last == escape {
						entry.WriteRune(r)
						return nil
					}
					if inQuote {
						d.results = append(d.results, entry.String())
						entry.Reset()
					}
					inQuote = !inQuote
					return nil
				case r == sep:
					if inQuote {
						entry.WriteRune(r)
						return nil
					}
					if lastNonSpace == q {
						return nil
					}
					if unicode.IsSpace(last) {
						if lastNonSpace == q {
							return nil
						}
					}
					return fmt.Errorf("found list entry separator(%s) in a weird place", d.EntryQuote)
				case r == rc:
					if read == 1 {
						return fmt.Errorf("the list close character(%s) cannot be the first character in a list", d.RightConstraint)
					}
					if inQuote {
						entry.WriteRune(r)
						return nil
					}
					if reader.Len() > 0 {
						x, _, _ := reader.ReadRune()
						return fmt.Errorf("cannot close a list with %s", string([]rune{r, x}))
					}
					foundRightConstraint = true
					return nil
				default:
					entry.WriteRune(r)
					return nil
				}
			}()
			if err != nil {
				return nil, err
			}
		}
	}
}
