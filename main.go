package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	signal.Ignore(syscall.SIGINT)
	os.Setenv("SHELL", "os-curriculum-project-shell")

	for {
		s, ok := r()
		if !ok {
			return
		}
		for _, s := range strings.Split(s, ";") {
			if s == "exit" {
				return
			}
			os.Setenv("?", Table.LastRun)

			s = os.ExpandEnv(s)
			ss := strings.Fields(s)
			if len(ss) == 0 {
				continue
			}

			var bg bool
			if ss[len(ss)-1] == "&" {
				ss = ss[:len(ss)-1]
				bg = true
			}

			out := e(ss, bg)
			p(out, bg)
		}
	}
}
