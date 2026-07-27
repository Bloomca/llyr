package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

var ErrCancelled = errors.New("prompt cancelled")

func selectOption(options []string) (int, error) {
	fd := int(os.Stdin.Fd())

	if !term.IsTerminal(fd) {
		return 0, errors.New("stdin is not a terminal")
	}

	// we need to make terminal raw so we can highlight the selected option
	old, err := term.MakeRaw(fd)
	if err != nil {
		return 0, err
	}
	defer term.Restore(fd, old)

	// save it to a variable so we can switch to Stderr if needed
	out := os.Stdout
	fmt.Fprint(out, "\x1b[?25l")       // hide cursor
	defer fmt.Fprint(out, "\x1b[?25h") // restore cursor on exit

	fmt.Fprintf(out, "%s\r\n", "Choose a tool:")

	sel := 0

	draw := func() {
		for i, opt := range options {
			line := " " + opt
			if i == sel {
				// highlight currently selected line in cyan
				line = "\x1b[36m> " + opt + "\x1b[0m"
			}
			fmt.Fprintf(out, "\x1b[2K%s\r\n", line)
		}
	}
	rewind := func() {
		// this moves cursor up several lines
		fmt.Fprintf(out, "\x1b[%dA", len(options))
	}

	draw()

	in := bufio.NewReader(os.Stdin)

	for {
		b, err := in.ReadByte()
		if err != nil {
			return 0, err
		}

		switch b {
		case '\r', '\n':
			return sel, nil
		case 27:
			// lone espace, pressed, just quit the application
			if in.Buffered() == 0 {
				return 0, ErrCancelled
			}
			b2, err := in.ReadByte()
			if err != nil {
				return 0, err
			}
			// '[' is CSI; 'O' is SS3, which some terminals send in
			// application cursor mode
			if b2 != '[' && b2 != 'O' {
				continue
			}
			b3, _ := in.ReadByte()
			switch b3 {
			case 'A':
				sel = (sel - 1 + len(options)) % len(options)
			case 'B':
				sel = (sel + 1) % len(options)
			default: // ignore every other sequence
				continue
			}
		default:
			continue
		}

		rewind()
		draw()
	}
}
