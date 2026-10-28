package main

import (
	"fmt"
	"io"
	"strings"
)

type Decoration int

const (
	DecorationNone = iota
	DecorationPlain
	DecorationFancy
)

const (
	colorRed = 31
	// ANSI colors -- using 32 - 36
	colorStart = 32
	numColors  = 5
)

type OutMsg struct {
	reflexID   int
	reflexName string
	msg        string
	clear      bool
}

const clearScreen = "\x1b[2J\x1b[3J\x1b[H"

func infoPrintln(id int, name string, args ...interface{}) {
	stdout <- OutMsg{reflexID: id, reflexName: name, msg: strings.TrimSpace(fmt.Sprintln(args...))}
}

func infoPrintf(id int, name string, format string, args ...interface{}) {
	stdout <- OutMsg{reflexID: id, reflexName: name, msg: fmt.Sprintf(format, args...)}
}

func printMsg(msg OutMsg, writer io.Writer) {
	var b strings.Builder
	if msg.clear {
		b.WriteString(clearScreen)
		if msg.msg == "" {
			writeLine(writer, b.String())
			return
		}
	}

	tag := ""
	if decoration == DecorationFancy || decoration == DecorationPlain {
		switch {
		case msg.reflexID < 0:
			tag = "[info]"
		case msg.reflexName != "":
			tag = "[" + msg.reflexName + "]"
		default:
			tag = fmt.Sprintf("[%02d]", msg.reflexID)
		}
	}

	if decoration == DecorationFancy {
		color := (msg.reflexID % numColors) + colorStart
		if msg.reflexID < 0 {
			color = colorRed
		}
		fmt.Fprintf(&b, "\x1b[01;%dm%s ", color, tag)
	} else if decoration == DecorationPlain {
		b.WriteString(tag + " ")
	}
	b.WriteString(msg.msg)
	if decoration == DecorationFancy {
		b.WriteString("\x1b[m")
	}
	if !strings.HasSuffix(msg.msg, "\n") {
		b.WriteByte('\n')
	}
	writeLine(writer, b.String())
}

// writeLine writes one complete decorated line. A terminal write error has
// nowhere to be reported, so it is intentionally dropped.
func writeLine(writer io.Writer, line string) {
	_, _ = io.WriteString(writer, line)
}

func printOutput(out <-chan OutMsg, outWriter io.Writer) {
	for msg := range out {
		printMsg(msg, outWriter)
	}
}
