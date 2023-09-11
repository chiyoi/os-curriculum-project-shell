package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
)

func r() (s string, ok bool) {
	u, e1 := user.Current()
	h, e2 := os.Hostname()
	p, e3 := os.Getwd()
	if e1 != nil || e2 != nil || e3 != nil {
		panic(fmt.Sprint(e1, e2, e3))
	}
	fmt.Printf("[%s@%s %s]$ ", u.Name, h, p)

	sc := bufio.NewScanner(os.Stdin)
	ok = sc.Scan()
	s = sc.Text()
	return
}

func e(ss []string, bg bool) (out chan string) {
	out = make(chan string, 1)

	cmd := exec.Command(ss[0], ss[1:]...)
	cmd.Stdout = ChanWriter(out)
	cmd.Stderr = ChanWriter(out)

	run := func() {
		defer close(out)
		if err := cmd.Run(); err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				out <- fmt.Sprintf("%s: command not found:%s\n", os.Getenv("SHELL"), ss[0])
			}
			return
		}
	}

	if bg {
		go run()
	} else {
		run()
	}
	return
}

func p(out chan string, bg bool) {
	if out == nil {
		return
	}

	pr := func() {
		for s := range out {
			fmt.Print(s)
		}
	}

	if bg {
		go pr()
	} else {
		pr()
	}
}

type ChanWriter chan string

func (cw ChanWriter) Write(p []byte) (n int, err error) {
	select {
	case cw <- string(p):
		return len(p), nil
	default:
		return 0, errors.New("channel unavailable")
	}
}
