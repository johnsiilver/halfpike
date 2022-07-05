/*
Package halfpike provides a lexer/parser framework library that can simplify lexing and parsing by using a very limited
subset of the regexp syntax. This prevents many of the common errors encountered when trying to parse output from devices
where the complete language syntax is unknown and can change between releases.  Routers and other devices with human
readable output or badly mangled formats within a standard (such as XML or JSON).

Called halfpike, because this solution is a mixture of Rob Pike's lexer talk and the use of regex's within a single line
of output to do captures in order to store a value within a struct type.

A similar method replaced complex regex captures at a large search company's network group to prevent accidental empty
matches and other bad behavior from regexes that led to issues in automation stacks.  It allowed precise diagnosis of
problems and readable code (complex regexes are not easily readable).
*/
package halfpike

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// stateFn is used to process some part of an input line either emitting tokens and
// returning the next stateFn or nil if terminating.
// The last token should be ItemEOL.
type stateFn func(l *lexer) stateFn

// ItemType describes the type of item being emitted by the Lexer. There are predefined ItemType(s)
// and the rest are defined by the user.
type ItemType int

const (
	// ItemUnknown indicates that the Item is an unknown. This should only happen on
	// a Item that is the zero type.
	ItemUnknown ItemType = iota
	// ItemEOF indicates that the end of input is reached. No further tokens will be sent.
	ItemEOF
	// ItemText indicates that it is a block of text separated by some type of space (including tabs).
	// This may contain numbers, but if it is not a pure number it is contained in here.
	ItemText
	// ItemInt indicates that an integer was found.
	ItemInt
	// ItemFloat indicates that a float was found.
	ItemFloat
	// ItemEOL indicates the end of a line was reached.
	ItemEOL
	// itemSpace indicates a space character as recognized by unicode.IsSpace().
	// This is private because our lexer does not emit these as they are unnecesary.
	itemSpace
)

// Line represents a line in the input.
type Line struct {
	// Items are the Item(s) that make up a line.
	Items []Item
	// LineNum is the line number in the content this represents, starting at 1.
	LineNum int
	// Raw is the actual raw string that made up the line.
	Raw string
}

// Item represents a token created by the Lexer.
type Item struct {
	// Type is the type of item that is stored in .Val.
	Type ItemType
	// Val is the value of the item that was in the text output.
	Val string

	// !!!!!The following fields are only output on an ItemEOL or ItemEOF.!!!!!

	// lineNum is the line number this item was found on.
	lineNum int

	// raw is the raw string for a line.
	raw string
}

// IsZero indicates the Item is the zero value.
func (i Item) IsZero() bool {
	return reflect.ValueOf(i).IsZero()
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

// ItemJoin takes a line, the inclusive beginning index and the non-inclusive ending index and
// joins all the values with a single space between them. -1 for start or end means from the absolute
// begin or end of the line slice. This will automatically remove the carriage return or EOF items.
func ItemJoin(line Line, start, end int) string {
	var l Line

	switch {
	case start == -1 && end == -1:
		l = line
	case start == -1:
		l.Items = line.Items[:end]
	case end == -1:
		l.Items = line.Items[start:]
	default:
		l.Items = line.Items[start:end]
	}

	b := strings.Builder{}
	for _, i := range l.Items {
		if i.Type == ItemEOL || i.Type == ItemEOF || i.Type == itemSpace {
			break
		}
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(i.Val)
	}
	return b.String()
}

// Lexer holds the state of the scanner.
type lexer struct {
	ctx context.Context

	input   string    // the string being scanned.
	start   int       // start position of this item.
	pos     int       // current position in the input.
	width   int       // width of last rune read from input.
	items   chan Item // channel of scanned items.
	startFn stateFn
}

// newLexer is the constructor for Lexer.
func newLexer(ctx context.Context, s string, start stateFn) *lexer {
	if start == nil {
		panic("start cannot be nil")
	}

	return &lexer{ctx: ctx, input: s, items: make(chan Item, 10), startFn: start}
}

// Reset resets the Lexer lex argument "s".
func (l *lexer) reset(s string) {
	l.input = s
	l.start = 0
	l.pos = 0
	l.width = 0
	l.items = make(chan Item, 10)
}

// run lexes the input by executing state functions until the state is nil.
func (l *lexer) run() {
	for state := l.startFn; state != nil; {
		state = state(l)
	}
	close(l.items) // No more tokens will be delivered.
}

// emit creates an item for content from the last emit() until this point in the run.
func (l *lexer) emit(t ItemType, ri ...rawInfo) ItemType {
	var item Item
	switch t {
	case ItemEOL, ItemEOF:
		item = Item{t, l.input[l.start:l.pos], ri[0].num, ri[0].str}
	default:
		item = Item{Type: t, Val: l.input[l.start:l.pos]}
	}
	l.addItemsChannel(item)
	l.start = l.pos
	return t
}

func (l *lexer) addItemsChannel(item Item) {
	select {
	case <-l.ctx.Done():
		// This simply causes the lexer to continue and finish off the content without
		// blocking on any channel.
		return
	case l.items <- item:
	}
}

// current shows what is currently stored in our buffer to be sent on the next emit().
func (l *lexer) current() string {
	if l.start >= len(l.input) || l.start == l.pos {
		return ""
	}

	return l.input[l.start:l.pos]
}

// ignore skips over the pending input before this point, meaning it wil not be used in an
// Item when Emit() is called.
func (l *lexer) ignore() {
	l.start = l.pos
}

// backup steps back one rune. Can be called only once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return rune(ItemEOF)
	}
	var r rune
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

type rawInfo struct {
	str string
	num int
}

func untilSpace(l *lexer) stateFn {
	lineNum := 0
	raw := strings.Builder{}

	last := ItemUnknown
	r := l.next()
	for ; true; r = l.next() {
		raw.WriteRune(r)

		switch {
		case r == '\n':
			switch last {
			// We don't care about blank lines.
			case ItemUnknown, ItemEOL:
				l.ignore()
				lineNum++
				continue
			}
			l.backup() // backup before the carriage return.
			switch {
			case isInt(l.current()):
				l.emit(ItemInt)
			case isFloat(l.current()):
				l.emit(ItemFloat)
			case last == itemSpace:
				// do nothing
			default:
				l.emit(ItemText)
			}

			// Emit the carriage return.
			l.next()
			last = l.emit(ItemEOL, rawInfo{raw.String(), lineNum})
			raw.Reset()

			lineNum++
		case r == rune(ItemEOF):
			l.emit(ItemEOF, rawInfo{raw.String(), lineNum})
			raw.Reset()
			return nil
		case unicode.IsSpace(r):
			switch last {
			case ItemUnknown, ItemEOL, itemSpace: // Ignore previous space characters
				l.ignore()
				continue
			}

			l.backup() // Remove the space.
			switch {
			case isInt(l.current()):
				l.emit(ItemInt)
			case isFloat(l.current()):
				l.emit(ItemFloat)
			default:
				l.emit(ItemText)
			}
			l.next() // Get ahead of the space
			l.ignore()
			last = itemSpace
		default:
			last = ItemText
		}
	}
	panic("untilSpace() unexpectantly escaped its 'for loop' without returning")
}

func isInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func isFloat(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// Validator provides methods to validate that a data type is okay.
type Validator interface {
	// Validate indicates if the type validates or not.
	Validate() error
}

// ParseFn handles parsing items provided by a lexer into an object that implements the Validator interface.
type ParseFn func(ctx context.Context, p *Parser) ParseFn

// ParseObject is an object that has a set of ParseFn methods, one of which is called Start()
// and a Validate() method. It is responsible for using the output of the Parser to turn the Items
// emitted by the lexer into structured data.
type ParseObject interface {
	Start(ctx context.Context, p *Parser) ParseFn
	Validator
}

// Parse starts a lexer that being sending items to a Parser instance. The function or method represented
// by "start" is called and passed the Parser instance to begin decoding into whatever form you want until
// a ParseFn returns ParseFn == nil.  If err == nil,
// the Validator object passed to Parser should have .Validate() called to ensure all data is correct.
func Parse(ctx context.Context, content string, parseObject ParseObject) error {
	p, err := newParser(content)
	if err != nil {
		return err
	}
	go p.lex.run()

	defer p.cancel()

	for state := parseObject.Start; state != nil; {
		state = state(ctx, p)
	}

	if err := p.HasError(); err != nil {
		return err
	}

	if err := parseObject.Validate(); err != nil {
		return err
	}
	return nil
}

// Parser parses items coming from the Lexer and puts the values into *struct that must satisfy the Validator interface.
// It provides helper methods for recording an Item directory to a field handling text conversions.  More complex types
// such as conversion to time.Time or custom objects are not covered. The Parser is created internally
// when calling the Parse() function.
type Parser struct {
	// ctx is simply used for cancelation and is not derived from anywhere.
	ctx    context.Context
	cancel context.CancelFunc

	lines []Line
	pos   int
	recv  chan Item

	lex       *lexer
	Validator Validator
	err       error
}

// newParser is the constructor for Parser.
func newParser(input string) (*Parser, error) {
	l := newLexer(context.Background(), input, untilSpace)

	ctx, cancel := context.WithCancel(context.Background())
	return &Parser{
		ctx:    ctx,
		cancel: cancel,
		lex:    l,
		recv:   l.items,
	}, nil
}

// Close closes the Parser. This must be called to prevent a goroutine leak.
func (p *Parser) Close() {
	p.cancel()
}

func (p *Parser) pull() Line {
	line := Line{}
	func() {
		for item := range p.recv {
			switch item.Type {
			case ItemEOF, ItemEOL:
				// The last Item records the raw and line value. Extract these from the item
				// and move them to the Line entries.
				line.Raw = item.raw
				line.LineNum = item.lineNum
				item.raw = ""
				item.lineNum = 0
				line.Items = append(line.Items, item)
			default:
				line.Items = append(line.Items, item)
				continue
			}
			return
		}
	}()
	return line
}

// HasError returns if the Parser encountered an error.
func (p *Parser) HasError() error {
	return p.err
}

// Errorf records an error in parsing. The ParseFn should immediately return nil.
// Errorf will always return a nil ParseFn.
func (p *Parser) Errorf(str string, args ...interface{}) ParseFn {
	p.err = fmt.Errorf(str, args...)
	return nil
}

// Reset will reset the Parsers internal attributes for parsing new input "s" into "val".
func (p *Parser) Reset(s string) error {
	p.lex.reset(s)
	p.lines = p.lines[:]
	p.recv = p.lex.items

	return nil
}

// Backup undoes a Next() call and returns the items in the previous line.
func (p *Parser) Backup() Line {
	p.pos--
	if p.pos < 0 {
		panic("parser.Backup() called on p.pos == 0")
	}
	return p.lines[p.pos]
}

// EOF returns true if the last Item in []Item is a ItemEOF.
func (p *Parser) EOF(line Line) bool {
	return line.Items[len(line.Items)-1].Type == ItemEOF
}

// Next moves to the next Line sent from the Lexer. That Line is returned. If we haven't
// received the next Line, the Parser will block until that Line has been received.
func (p *Parser) Next() Line {
	// We don't have any items, so grab the next item.
	if len(p.lines) == 0 {
		p.lines = append(p.lines, p.pull())
		p.pos = 1
		return p.lines[0]
	}

	// See if we already have found the end of input.
	if p.pos >= len(p.lines) {
		lastLine := len(p.lines) - 1
		if p.EOF(p.lines[lastLine]) {
			return p.lines[lastLine]
		}
	}

	// See if we are at the end of our slice and if so grab the next entry from the channel.
	if p.pos >= len(p.lines) {
		p.lines = append(p.lines, p.pull())
		p.pos++
		return p.lines[p.pos-1]
	}

	p.pos++
	return p.lines[p.pos-1]
}

// Peek returns the item in the next position, but does not change the current position.
func (p *Parser) Peek() Line {
	i := p.Next()
	p.Backup()
	return i
}

// Skip provides a special string for FindStart that will skip an item.
const Skip = "$.<skip>.$"

// FindStart looks for an exact match of starting items in a line represented by Line
// continuing to call .Next() until a match is found or EOF is reached.
// Once this is found, Line is returned. This is done from the current position.
func (p *Parser) FindStart(find []string) (Line, error) {
	for line := p.Next(); true; line = p.Next() {
		if p.IsAtStart(line, find) {
			return line, nil
		}

		if p.EOF(line) {
			return Line{}, fmt.Errorf("end of file reached without finding items: %#+v", find)
		}
	}
	panic("FindStart() escaped for loop without returning")
}

// FindUntil searches a Line until it matches "find", matches "until" or reaches the EOF. If "find" is
// matched, we return the Line. If "until" is matched, we call .Backup() and return true. This
// is useful when you wish to discover a line that represent a sub-entry of a record (find) but wish to
// stop searching if you find the beginning of the next record (until).
func (p *Parser) FindUntil(find []string, until []string) (matchFound Line, untilFound bool, err error) {
	for line := p.Next(); true; line = p.Next() {
		if p.IsAtStart(line, find) {
			return line, false, nil
		}
		if p.IsAtStart(line, until) {
			p.Backup()
			return Line{}, true, nil
		}

		if p.EOF(line) {
			return Line{}, false, fmt.Errorf("end of file reached without finding items: %#+v", find)
		}
	}
	panic("FindUntil() escaped for loop without returning")
}

// IsAtStart checks to see that "find" is at the beginning of "line".
func (p *Parser) IsAtStart(line Line, find []string) bool {
	if len(find) == 0 {
		return true
	}

	if len(line.Items) < len(find) {
		return false
	}

	for i, f := range find {
		if f == Skip {
			continue
		}

		if line.Items[i].Val != f {
			return false
		}
	}

	return true
}

// FindREStart looks for a match of [n]*regexp.Regexp against [n]Item.Val continuing to call .Next()
// until a match is found or EOF is reached. Once this is found, Line is returned. This is done from the current position.
func (p *Parser) FindREStart(find []*regexp.Regexp) (Line, error) {
	if len(find) == 0 {
		return Line{}, fmt.Errorf("cannot pass empty []*regexp.Regexp to FindREStart()")
	}

	for line := p.Next(); true; line = p.Next() {
		if p.IsREStart(line, find) {
			return line, nil
		}

		if p.EOF(line) {
			return Line{}, fmt.Errorf("FindStart: end of file reached without finding items: %#+v", find)
		}
	}
	panic("FindREStart() escaped for loop without returning")
}

// IsREStart checks to see that matches to "find" is at the beginning of "line".
func (p *Parser) IsREStart(line Line, find []*regexp.Regexp) bool {
	if len(find) == 0 {
		return false
	}

	if len(line.Items) < len(find) {
		return false
	}

	for i, f := range find {
		if !f.MatchString(line.Items[i].Val) {
			return false
		}
	}

	return true
}

// Match returns matches of the regex with keys set to the submatch names.
// If these are not named submatches (aka `(?P<name>regex)`) this will probably panic.
// A match that is empty string will cause an error to return.
func Match(re *regexp.Regexp, s string) (map[string]string, error) {
	names := re.SubexpNames()[1:]

	matches := re.FindStringSubmatch(s)
	if len(matches) < 1 {
		return nil, fmt.Errorf("")
	}

	matches = matches[1:]
	m := map[string]string{}

	for i, v := range matches {
		if v == "" {
			continue
		}
		m[names[i]] = v
	}
	if len(m) == 0 {
		return nil, fmt.Errorf("no matches were found")
	}

	return m, nil
}
