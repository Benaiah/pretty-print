package prettyPrintJson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

type Settings struct {
	Indent       bool
	Highlight    bool
	IndentAmount int
	Colors       Colors
}

type Colors struct {
	DelimFg  color.Attribute
	DelimBg  color.Attribute
	KeyFg    color.Attribute
	KeyBg    color.Attribute
	BoolFg   color.Attribute
	BoolBg   color.Attribute
	StringFg color.Attribute
	StringBg color.Attribute
	NumberFg color.Attribute
	NumberBg color.Attribute
}

var DefaultColors = Colors{
	DelimFg:  color.FgWhite,
	DelimBg:  color.BgBlack,
	KeyFg:    color.FgBlue,
	KeyBg:    color.BgBlack,
	BoolFg:   color.FgCyan,
	BoolBg:   color.BgBlack,
	StringFg: color.FgMagenta,
	StringBg: color.BgBlack,
	NumberFg: color.FgRed,
	NumberBg: color.BgBlack,
}

var DefaultSettings = Settings{
	Indent:       true,
	Highlight:    true,
	IndentAmount: 2,
	Colors:       DefaultColors,
}

func newline(dest io.Writer, indentLevel int) {
	fmt.Fprint(dest, "\n")
}

func writeline(dest io.Writer, line *bytes.Buffer, indentLevel int) {
	fmt.Fprintf(dest, "\n%v", strings.Repeat(" ", indentLevel))
	line.WriteTo(dest)
	line.Truncate(0)
}

// This type keeps track of whether we're in an array or not. It works
// as follows:
//
// - When we hit an opening object delimiter, we push `false`
// - When we hit an opening array delimiter, we push `true`
// - When we hit a closing delimiter, we pop
//
// This is needed in order to tell if an incoming string is a key or a
// value. Without a stack to keep track of it, arrays with nested
// structures would be impossible to keep track of.
type boolStack struct {
	s []bool
}

func newBoolStack() *boolStack {
	return &boolStack{s: make([]bool, 0)}
}

func (s *boolStack) push(b bool) {
	s.s = append(s.s, b)
}

func (s *boolStack) pop() bool {
	l := len(s.s)
	// We don't care about over-popping, as the json decoder will already
	// error out if there are mismatched tokens. When the array is empty,
	// we just return `false`
	if l == 0 {
		return false
	}

	res := s.s[l-1]
	s.s = s.s[:l-1]
	return res
}

func (s *boolStack) peek() bool {
	l := len(s.s)
	if l == 0 {
		return false
	}

	res := s.s[l-1]
	return res
}

func (s *boolStack) len() int {
	return len(s.s)
}

func Fprint(dest io.Writer, jsonInput []byte, settings Settings) error {
	// At the moment indentation and highlighting are all that the
	// function does, so if those are set to false it's just a noop
	if settings.Indent == false && settings.Highlight == false {
		return nil
	}

	// The decoder will decode the tokens one by one.
	dec := json.NewDecoder(bytes.NewReader(jsonInput))

	// Create the color writers
	writers := make(map[string]*color.Color)

	writers["delim"] = color.New(settings.Colors.DelimBg, settings.Colors.DelimFg)
	writers["key"] = color.New(settings.Colors.KeyBg, settings.Colors.KeyFg)
	writers["bool"] = color.New(settings.Colors.BoolBg, settings.Colors.BoolFg)
	writers["string"] = color.New(settings.Colors.StringBg, settings.Colors.StringFg)
	writers["number"] = color.New(settings.Colors.NumberBg, settings.Colors.NumberFg)

	indentLevel := 0
	isNextStringKey := true

	inArrayStack := newBoolStack()

	line := &bytes.Buffer{}

	// Loop through the tokens
	for {
		t, err := dec.Token()

		// When we hit EOF we're done, and can return the value
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch t.(type) {
		case json.Delim:
			// We need to check the value of the token to determine when we indent/dedent
			stringToken := fmt.Sprint(t)

			indentAfter := false

			writers["delim"].Fprintf(line, "%v", t)

			switch stringToken {
			case "{":
				isNextStringKey = true
				inArrayStack.push(false)
				indentAfter = true
			case "[":
				isNextStringKey = false
				inArrayStack.push(true)
				indentAfter = true
			case "}":
				fallthrough
			case "]":
				indentLevel -= 1
				inArrayStack.pop()
				isNextStringKey = !inArrayStack.peek()
				if inArrayStack.len() != 0 {
					fmt.Fprint(line, ",")
				}
			}

			writeline(dest, line, indentLevel*settings.IndentAmount)

			if indentAfter {
				indentLevel += 1
			}

		case string:
			if isNextStringKey {
				writers["key"].Fprintf(line, "\"%v\"", t)
				fmt.Fprint(line, ": ")
				isNextStringKey = false
			} else {
				writers["string"].Fprintf(line, "\"%v\"", t)
				fmt.Fprint(line, ",")
				isNextStringKey = !inArrayStack.peek()

				writeline(dest, line, indentLevel*settings.IndentAmount)
			}

		case float64:
			writers["number"].Fprint(line, t)
			fmt.Fprint(line, ",")
			isNextStringKey = !inArrayStack.peek()

			writeline(dest, line, indentLevel*settings.IndentAmount)

		case bool:
			writers["bool"].Fprint(line, t)
			fmt.Fprint(line, ",")
			isNextStringKey = !inArrayStack.peek()

			writeline(dest, line, indentLevel*settings.IndentAmount)

		default:
			fmt.Fprint(line, "WTF")
		}
	}

	return nil
}

func Print(jsonInput []byte, settings Settings) error {
	return Fprint(os.Stdout, jsonInput, settings)
}
