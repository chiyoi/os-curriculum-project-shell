package main

import (
	"os"
	"os/user"
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
