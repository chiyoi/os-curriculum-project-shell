package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"
)

func main() {
	u, err := user.Current()
	if err != nil {
		u = &user.User{
			Username: "Unknown",
			HomeDir:  "/",
		}
	}

	os.Setenv("SHELL", "os-curriculum-project-shell")

	var d Data
	d.LastRun = "0"

	for {
		l(u, &d)
	}
}

func l(u *user.User, d *Data) {
	s, ok := r(u)
	if !ok {
		os.Exit(0)
	}

	for _, s := range strings.Split(s, ";") {
		s = strings.ReplaceAll(s, "~", u.HomeDir)
		s = strings.ReplaceAll(s, "$?", d.LastRun)
		s = strings.ReplaceAll(s, "$*", strings.Join(os.Args[1:], " "))
		for i, a := range os.Args[1:] {
			s = strings.ReplaceAll(s, fmt.Sprintf("$%d", i), a)
		}
		s = os.ExpandEnv(s)

		ss := strings.Fields(s)
		if len(ss) == 0 {
			return
		}

		switch ss[0] {
		case "exit":
			var c int
			if len(ss) > 1 {
				if n, err := strconv.Atoi(ss[1]); err != nil {
					c = n
				}
			}
			os.Exit(c)
		case "cd":
			if len(ss) == 1 {
				os.Chdir(u.HomeDir)
				return
			}
			if len(ss) != 2 {
				fmt.Printf("%s: syntax error.\n", os.Getenv("SHELL"))
				return
			}

			if err := os.Chdir(ss[1]); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					fmt.Printf("%s: no such file or directory: %s\n", os.Getenv("SHELL"), ss[1])
				} else {
					fmt.Printf("%s: not a directory: %s\n", os.Getenv("SHELL"), ss[1])
				}
			}
			return
		}

		var rw io.ReadWriter
		cs := strings.Split(s, "|")
		for i, c := range cs {
			ss := strings.Fields(c)
			if len(ss) == 0 {
				fmt.Println()
			}

			var bg bool
			if ss[len(ss)-1] == "&" {
				if i != len(cs)-1 {
					fmt.Printf("%s: syntax error.", os.Getenv("SHELL"))
				}

				ss = ss[:len(ss)-1]
				bg = true
			}

			var rio []string
			for i := range ss {
				if ss[i] == "<" || ss[i] == ">" {
					ss, rio = ss[:i], ss[i:]
					break
				}
			}

			var in io.Reader
			var out io.Writer
			if rw != nil {
				in = rw
				rw = nil
			}

			for i := range rio {
				if rio[i] == "<" || rio[i] == ">" {
					if i+1 >= len(rio) {
						fmt.Printf("%s: syntax error.\n", os.Getenv("SHELL"))
						return
					}

					if rio[i] == "<" {
						f, err := os.Open(rio[i+1])
						if err != nil {
							fmt.Printf("%s: no such file or directory: %s\n", os.Getenv("SHELL"), rio[i+1])
							return
						}

						if in == nil {
							in = f
						} else {
							in = io.MultiReader(in, f)
						}
					} else {
						f, err := os.Create(rio[i+1])
						if err != nil {
							fmt.Printf("%s: open file failed: %s\n", os.Getenv("SHELL"), rio[i+1])
							return
						}

						if out == nil {
							out = f
						} else {
							out = io.MultiWriter(out, f)
						}
					}
				}
			}

			if in == nil {
				in = os.Stdin
			}

			if i+1 < len(cs) {
				rw = new(bytes.Buffer)
				if out == nil {
					out = rw
				} else {
					out = io.MultiWriter(out, rw)
				}
			}

			if out == nil {
				if bg {
					c := make(chan string, 1)
					rw = ChannelRW(c)
				} else {
					rw = new(bytes.Buffer)
				}
				out = rw
			}

			e(ss, bg, in, out, d)
			p(rw, bg)
		}
	}
}

type ChannelRW chan string

func (cw ChannelRW) Write(p []byte) (n int, err error) {
	select {
	case cw <- string(p):
		return len(p), nil
	default:
		return 0, errors.New("channel unavailable")
	}
}

func (cw ChannelRW) Read(p []byte) (n int, err error) {
	s, ok := <-cw
	if ok {
		return copy(p, s), nil
	}
	return 0, io.EOF
}
