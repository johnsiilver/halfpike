# HalfPike - The Helpful Lexing/Parsing module

![gopherfs logo-sm](https://raw.githubusercontent.com/gopherfs/fs/main/cover.png)

[![GoDoc](https://godoc.org/github.com/gopherfs/fs?status.svg)](https://godoc.org/github.com/gopherfs/fs)
[![Go Report Card](https://goreportcard.com/report/github.com/johnsiilver/halfpike)](https://goreportcard.com/report/github.com/johnsiilver/halfpike)

Since you've made it this far, why don't you hit that :star: up in the right corner.

## Introduction

Halfpike provides a package that handles lexing for you so that you can parse textual output using a set of parsing tools we provide. This is how a language compiler turns your text into something it can use.

This technique is much less error prone than trying to use Regexes.

This can be used to convert textual output into structured output software can use. This has been used to parse router output into structs and protocol buffers and to convert an IDL language into data structures.

## History

Halfpike was originally written at Google to support lexing/parsing router configurations from vendor routers (Juniper, Cisco, Foundry, Brocade, Force10, ...) that had at least some information  only in human readble forms. This required we parse the output from the router's into structured data. The led to various groups writing complicated Regex expressions for each niche use case, which didn't scale well.

Regexes were eventually replaced with something called TextFSM. And while it was a great improvement over what we had, it relied heavily on Regexes. This led to difficult debugging when we ran into a problem and had a tendency to have Regex writers provide loose Regexes that put a zero value for a type, like 0 for an integer, when the router vendor would change output formats between versions. That caused a few router configuration issues in production. It also required its own special debugger to debug problems.

Halfpike was written to be used in an intermediate service that converted textual output from the router into protocol buffers for services to consume. It improved on TextFSM by erroring on anything that it does not understand. Debugging with Halfpike is also greatly simplified. 

But nothing is free, Halfpike parsing is complex to write and the concepts take longer to understand. At its core, it provides helpful utilities around the lexing/parsing techniques that language compilers use. My understanding is that the maintainers of services based on Halfpike at Google hate when they need to do something with it, but don't replace it because is provides a level of safety that is otherwise hard to achieve.

This version differs from the Google one in that it is a complete re-write (as I didn't have access to the source after I left) from what I remember, and so this is likely to differ in major ways. But it is built on the same concepts.

It is also going to differ in that I have used this code to write my own Intermediate Description Language (IDL), similiar to .proto files. So I have expanded what the code can do to help me in that use case.

The name comes from Rob Pike, who gave a talk on lexical scanning in Go: https://www.youtube.com/watch?v=HxaD_trXwRE .  As this borrows a lot of concepts from this while providing some limited helpful tooling that use things like Regexes (in a limited capacity), I say its about half of what he was talking about. Rob certainly does not endorse this project.

## Concepts

HalfPike is line oriented, in that it scans a file and returns line items. You use those sets of items to decicde how you want to decode a line. HalfPike ignores lines that only have space characters and ignores spaces between line items. 

Let's say we wanted to decode the "package" line inside a Go file:

```go
package mypackage
```

We want to decode that into a struct that represents a file:

```go
type File struct {
    Package string
}
```

And to make sure that after we finish decoding, everything is set to the right value. It will implement `halfpike.Validator`:

```go
func (f *File) Validate() error {
    if f.Package == "" {
        return fmt.Errorf("every Go file must have a 'package' declaration")
    }
    // We could add deeper checks, such as that it starts with a lower case letter
    // and only contains certain characters. We could also check that directly in our parser.
    return nil
}
```

Let's create a function that is used to start parsing. We must create at least one and it must have the name `Start` and it will implement `halfpike.ParseFn`:

```go
func (f *File) Start(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
    // Simply passes us to another halfpike.ParseFn that handles the 'package' line.
    return f.parsePackage 
}
```

Now we will implement the package line parser:

```go
func (f *File) parsePackage(ctx context.Context, p *halfpike.Parser) halfpike.ParseFn {
    // This gets the first line of the file. We skip any blank lines.
    // p.Next() will always return a line. If there are no more lines, it returns the
    // last line again. You can check if the line is the last line with p.EOF().
    line := p.Next() 

    if len(line.Items) != 3 { // 'package' keyword + package name + EOL or EOF item
        // Parser.Errorf() records an error in the Parser and returns a nil halfpike.ParseFn, 
        // which tells the Parser to stop parsing.
        return p.Errorf("[Line %d] first line of file must be the 'package' line and must contain a package name", line.LineNum)
    }

    if line.Items[0].Val != "package" {
        if strings.ToLower(line.Items[0].Val) == "package" {
            return p.Errorf("[Line %d] 'package' keyword found, but had wrong case", line.LineNum)
        }
        return p.Errorf("[Line %d] expected first word to be 'package', found %q", line.LineNu, line.Items[0].Val)
    }

    if line.Items[1].Type != halfpike.ItemText {
        return p.Errorf("[Line %d] 'package' keyword should be followed by a valid package name", line.LineNum)
    }
    f.Package = line.Items[1].Val

    // Make sure the end is either EOL or EOF. It is also trivial to look for and remove
    // line comments.
    switch line.Items[2].Type {
    case halfpike.ItemEOL, halfpike.ItemEOF:
    default:
        return p.Errorf("[Line %d] 'package' statement had unsupported end item, %q", line.LineNum, line.Items[2].Val)
    }

    // If we return nil, the parsing ends. If we return another ParseFn method, it will be executed.
    // Once our execution stops, the Validate() we defined gets executed.
    return nil
}
```

Executing our parser against our file content is simple:

```go
    ctx := context.Background()
	f := &File{}

	// Parses our content in showBGPNeighbor and begins parsing with BGPNeighbors.Start().
	if err := halfpike.Parse(ctx, fileContent, f); err != nil {
		panic(err)
	}
```

You can give this a try at: https://go.dev/play/p/iFGRIM3Ho_z

This is a simple example of parsing a file. You can easily see this takes much more work than simply using Regexes. And for something this simple, it would be crazy to use HalfPike. But if you have a more complicated input to deconstruct that can't have errors in the parsing, HalfPike can be helpful, if somewhat verbose.

## Advanced Features

The above section simply covers the basics. We offer more advanced tools such as:

### `Parser.FindStart()`

This will search for a line with a list of Item values that a line must match. This allows you to skip over lines you don't care about (often handy if you need just a subset of information). 

We allow you to use the `Skip` value to skip over Items in a line that don't need to match.  

Say you were looking through output of time values for a line that had "Time Now: 12:53:04 UTC". Clearly you cannot match on "12:53:04", as it will change every time you run the output. So you can provide: `[]string{"Time", "Now:", halfpike.Skip, "UTC"}`.


### `Parser.FindUntil()`

In the same veing as `FindStart()`, this function is useful for searching through sub-entries of an parent entry, but stopping if you find a new parent entry.

### `Parser.IsAtStart()`

Checks to see if a line starts with some items.

### `Parser.FindREStart()`

Like `Parser.FindStart()` but with Regexes!

### `Parser.IsREStart()`

Checks that the regexes passed match the Items in the same position in a line. If they do, it returns true.

### `Parser.Match()`

Allows passing a `regexp.Regexp` against a string (like `line.Raw`) to extract matches into a `map[string]string`. This requires a Regexp that uses named submatches (like `(?P<name>regex)`) in order to work.

## The `line` Package

Sometimes we want to disect a line with the whitespaces included and need some more advanced features. There is a separate `line` package that contains a `Lexer` that will return all parts of a line, including whitespace. 

It also includes an `Item` type that can answer many more questions about an `Item`. Here are a few of the methods it contains:s

* HasPrefix()
* HasSuffix()
* Capitalized() 
* StartsWithLetter()
* OnlyLetters()
* OnlyLettersAndNumbers()
* OnlyHas()
* ContainsNumbers()
* ASCIIOnly()

And for something to handle reading those pesky lists of items:

* DecodeList{}

## More examples

The GoDoc itself contains two examples: a "short" and "long" example.  These are both based on parsing router configuration and they are complex.  

You can find the IDL parser that uses HalfPike I wrote for the Claw encoding format here:
https://github.com/bearlytools/claw/tree/main/internal/idl
