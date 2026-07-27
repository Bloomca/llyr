package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"
)

func splitArguments(line string) ([]string, error) {
	const (
		bare = iota
		inSingle
		inDouble
	)

	var (
		args  []string
		buf   strings.Builder
		open  bool
		state = bare
	)

	flush := func() {
		if open {
			args = append(args, buf.String())
			buf.Reset()
			open = false
		}
	}

	rs := []rune(line)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		switch state {
		case bare:
			switch {
			case unicode.IsSpace(c):
				flush()
			case c == '\'':
				open, state = true, inSingle
			case c == '"':
				open, state = true, inDouble
			case c == '\\':
				if i+1 > len(rs) {
					return nil, errors.New("trailing backslash")
				}
				i++
				buf.WriteRune(rs[i])
				open = true
			default:
				buf.WriteRune(c)
				open = true
			}
		case inSingle:
			if c == '\'' {
				state = bare
			} else {
				buf.WriteRune(c)
			}
		case inDouble:
			switch c {
			case '"':
				state = bare
			case '\\':
				// Only these four are escapable inside double quotes; before
				// anything else the backslash is a literal character.
				if i+1 < len(rs) && strings.ContainsRune(`"\$`+"`", rs[i+1]) {
					i++
				}
				buf.WriteRune(rs[i])
			default:
				buf.WriteRune(c)
			}
		}
	}

	if state != bare {
		return nil, errors.New("unterminated quote")
	}
	flush()

	if len(args) == 0 {
		return nil, errors.New("empty command")
	}

	return args, nil
}

func executeCommand(dir string, line string, prompt string) *exec.Cmd {
	args, err := splitArguments(line)
	args = append(args, prompt)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	path, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Printf("resolving %q: %w", args[0], err)
	}

	cmd := exec.Command(path, args[1:]...)
	cmd.Dir = dir

	return cmd
}
