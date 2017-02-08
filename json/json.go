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
	Highlight:    true,
	IndentAmount: 2,
	Colors:       DefaultColors,
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

type colorWriter struct {
	writers         map[string]*color.Color
	shouldHighlight bool
}

func newColorWriter(colors Colors, shouldHighlight bool) colorWriter {
	// Create the color writers
	writers := make(map[string]*color.Color)

	writers["delim"] = color.New(colors.DelimBg, colors.DelimFg)
	writers["key"] = color.New(colors.KeyBg, colors.KeyFg)
	writers["bool"] = color.New(colors.BoolBg, colors.BoolFg)
	writers["string"] = color.New(colors.StringBg, colors.StringFg)
	writers["number"] = color.New(colors.NumberBg, colors.NumberFg)

	return colorWriter{writers, shouldHighlight}
}

func (c *colorWriter) Fprintf(writer string, dest io.Writer, format string, v ...interface{}) {
	if c.shouldHighlight {
		c.writers[writer].Fprintf(dest, format, v...)
	} else {
		fmt.Fprintf(dest, format, v...)
	}
}

func (c *colorWriter) Fprint(writer string, dest io.Writer, v interface{}) {
	if c.shouldHighlight {
		c.writers[writer].Fprint(dest, v)
	} else {
		fmt.Fprint(dest, v)
	}
}

func Fprint(dest io.Writer, jsonInput []byte, settings Settings) error {
	writer := newColorWriter(settings.Colors, settings.Highlight)

	// The decoder will decode the tokens one by one.
	dec := json.NewDecoder(bytes.NewReader(jsonInput))

	// We need to encode strings so they're escaped properly
	encodingBuffer := &bytes.Buffer{}
	enc := json.NewEncoder(encodingBuffer)

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

			writer.Fprintf("delim", line, "%v", t)

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
			strVal := fmt.Sprint(t)
			enc.Encode(strVal)
			str := strings.TrimSpace(encodingBuffer.String())
			encodingBuffer.Reset()

			if isNextStringKey {
				writer.Fprint("key", line, str)
				fmt.Fprint(line, ": ")
				isNextStringKey = false
			} else {
				writer.Fprint("string", line, str)
				fmt.Fprint(line, ",")
				isNextStringKey = !inArrayStack.peek()

				writeline(dest, line, indentLevel*settings.IndentAmount)
			}

		case float64:
			writer.Fprint("number", line, t)
			fmt.Fprint(line, ",")
			isNextStringKey = !inArrayStack.peek()

			writeline(dest, line, indentLevel*settings.IndentAmount)

		case bool:
			writer.Fprint("bool", line, t)
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
