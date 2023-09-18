package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
)

func r(u *user.User) (s string, ok bool) {
	h, e1 := os.Hostname()
	p, e2 := os.Getwd()
	if e1 != nil || e2 != nil {
		panic(fmt.Sprint(e1, e2))
	}
	p = strings.ReplaceAll(p, u.HomeDir, "~")
	fmt.Printf("[%s@%s %s]$ ", u.Username, h, p)

	sc := bufio.NewScanner(os.Stdin)
	ok = sc.Scan()
	s = sc.Text()
	return
}

func e(ctx context.Context, ss []string, in io.Reader, out io.Writer, d *Data) {
	cmd := exec.CommandContext(ctx, ss[0], ss[1:]...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out

	defer func() {
		if c, ok := out.(ChannelRW); ok {
			close(c)
		}
	}()

	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			fmt.Fprintf(out, "%s: command not found:%s\n", os.Getenv("SHELL"), ss[0])
		} else if ee := (*exec.ExitError)(nil); errors.As(err, &ee) {
			d.LastRun = strconv.Itoa(ee.ProcessState.ExitCode())
		} else {
			panic(err)
		}
		return
	}
	d.LastRun = "0"
}

func p(r io.Reader) {
	if r == nil {
		return
	}

	if _, err := io.Copy(os.Stdout, r); err != nil {
		fmt.Printf("%s: unknown error.", os.Getenv("SHELL"))
		return
	}
}

func l(u *user.User, d *Data) {
	s, ok := r(u)
	if !ok {
		os.Exit(0)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer func() {
		stop()
		signal.Ignore(os.Interrupt)
	}()
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

		var bg bool
		if ss[len(ss)-1] == "&" {
			bg = true
		}

		var rw io.ReadWriter
		cs := strings.Split(s, "|")
		for i, c := range cs {
			ss := strings.Fields(c)
			if len(ss) == 0 {
				fmt.Printf("%s: syntax error.\n", os.Getenv("SHELL"))
				return
			}

			if ss[len(ss)-1] == "&" {
				ss = ss[:len(ss)-1]
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
						in = tr(in, f)
					} else {
						f, err := os.Create(rio[i+1])
						if err != nil {
							fmt.Printf("%s: open file failed: %s\n", os.Getenv("SHELL"), rio[i+1])
							return
						}
						out = tw(out, f)
					}
				}
			}

			if in == nil {
				in = os.Stdin
			}

			if i+1 < len(cs) || out == nil {
				rw = MakeChannelRW(0)
				if out == nil {
					out = rw
				} else {
					out = io.MultiWriter(out, rw)
				}
			}

			go e(ctx, ss, in, out, d)
		}

		if bg {
			go p(rw)
		} else {
			p(rw)
		}
	}
}

func tr(r, r1 io.Reader) io.Reader {
	if r != nil {
		return r1
	}
	return io.MultiReader(r, r1)
}

func tw(w, w1 io.Writer) io.Writer {
	if w == nil {
		return w1
	}
	return io.MultiWriter(w, w1)
}

type ChannelRW chan []byte

func MakeChannelRW(n int) ChannelRW {
	return make(chan []byte, n)
}

func (cw ChannelRW) Write(p []byte) (n int, err error) {
	select {
	case cw <- p:
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