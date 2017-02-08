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
