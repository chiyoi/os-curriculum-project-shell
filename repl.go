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

func e(ss []string, in io.Reader, out, err io.Writer, d *Data) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer func() {
		stop()
		signal.Ignore(os.Interrupt)
	}()

	cmd := exec.CommandContext(ctx, ss[0], ss[1:]...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = err

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

func l(d *Data) {
	// Get necessary environment.
	u, err := user.Current()
	if err != nil {
		u = &user.User{
			Username: "Unknown",
			HomeDir:  "/",
		}
	}

	// Wait for input.
	s, ok := r(u)
	if !ok {
		os.Exit(0)
	}

	// Resolve and execute command.
	for _, s := range strings.Split(s, ";") {
		// Expend variables.
		s = strings.ReplaceAll(s, "~", u.HomeDir)
		s = os.Expand(s, func(ss string) string {
			switch ss {
			case "?":
				return d.LastRun
			case "*", "@":
				return strings.Join(os.Args[1:], " ")
			case "#":
				return strconv.Itoa(len(os.Args) - 1)
			}

			i, err := strconv.Atoi(ss)
			if err == nil && i < len(os.Args) {
				return os.Args[i]
			}
			return os.Getenv(ss)
		})

		// Resolve Builtin commands.
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

		// Flag Background.
		var bg bool
		if ss[len(ss)-1] == "&" {
			bg = true
		}

		// Resolve Redirection.
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
				if RedirectionResolver(ss[i]) != nil {
					ss, rio = ss[:i], ss[i:]
					break
				}
			}

			var in io.Reader
			var out, err io.Writer
			if rw != nil {
				in = rw
			}

			for i := range rio {
				if resolve := RedirectionResolver(rio[i]); resolve != nil {
					var f string
					if i+1 < len(rio) {
						f = rio[i+1]
					}
					if resolve == nil || !resolve(f, &in, &out, &err) {
						return
					}
				}
			}

			if in == nil {
				in = os.Stdin
			}
			if i == len(cs)-1 && out != nil {
				rw = nil
			} else {
				rw = MakeChannelRW(0)
				out = tw(out, rw)
			}
			if err == nil {
				err = os.Stderr
			}

			// Run command.
			go e(ss, in, out, err, d)
		}

		// Print result.
		if bg {
			go p(rw)
		} else {
			p(rw)
		}
	}
}
