package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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

func e(ss []string, bg bool, in io.Reader, out io.Writer, d *Data) {
	cmd := exec.Command(ss[0], ss[1:]...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = os.Stdout

	run := func() {
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

	if bg {
		go run()
	} else {
		run()
	}
}

func p(r io.Reader, bg bool) {
	if r == nil {
		return
	}

	pr := func() {
		bs, err := io.ReadAll(r)
		if err != nil {
			fmt.Printf("%s: unknown error.", os.Getenv("SHELL"))
			return
		}
		fmt.Print(string(bs))
	}

	if bg {
		go pr()
	} else {
		pr()
	}
}
