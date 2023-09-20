package main

import (
	"os"
	"os/signal"
)

func main() {
	var d Data
	d.LastRun = "0"
	signal.Ignore(os.Interrupt)
	os.Setenv("SHELL", "os-curriculum-project-shell")
	for {
		l(&d)
	}
}

type Data struct {
	LastRun string
}
