package prettyPrintJson

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

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
