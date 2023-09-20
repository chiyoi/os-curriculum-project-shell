package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func RedirectionResolver(symbol string) func(f string, in *io.Reader, out, er *io.Writer) (ok bool) {
	switch symbol {
	case ">", "1>":
		return func(f string, in *io.Reader, out, er *io.Writer) (ok bool) {
			fd, err := os.Create(f)
			if err != nil {
				fmt.Printf("%s: open file failed: %s\n", os.Getenv("SHELL"), f)
				return
			}
			ok = true
			*out = tw(*out, fd)
			return
		}
	case ">>", "1>>":
		return func(f string, in *io.Reader, out, er *io.Writer) (ok bool) {
			fd, err := os.OpenFile(f, os.O_RDWR|os.O_APPEND, 0)
			if err != nil {
				fmt.Printf("%s: open file failed: %s\n", os.Getenv("SHELL"), f)
				return
			}
			ok = true
			*out = tw(*out, fd)
			return
		}
	case "<", "0<":
		return func(f string, in *io.Reader, out, er *io.Writer) (ok bool) {
			fd, err := os.Open(f)
			if err != nil {
				fmt.Printf("%s: no such file or directory: %s\n", os.Getenv("SHELL"), f)
				return
			}
			ok = true
			*in = tr(*in, fd)
			return
		}
	case "2>":
		return func(f string, in *io.Reader, out, er *io.Writer) (ok bool) {
			fd, err := os.Create(f)
			if err != nil {
				fmt.Printf("%s: open file failed: %s\n", os.Getenv("SHELL"), f)
				return
			}
			ok = true
			*er = tw(*er, fd)
			return
		}
	case "2>>":
		return func(f string, in *io.Reader, out, er *io.Writer) (ok bool) {
			fd, err := os.OpenFile(f, os.O_RDWR|os.O_APPEND, 0)
			if err != nil {
				fmt.Printf("%s: open file failed: %s\n", os.Getenv("SHELL"), f)
				return
			}
			ok = true
			*er = tw(*er, fd)
			return
		}
	case "&>":
		return func(f string, in *io.Reader, out, er *io.Writer) (ok bool) {
			fd, err := os.Create(f)
			if err != nil {
				fmt.Printf("%s: open file failed: %s\n", os.Getenv("SHELL"), f)
				return
			}
			ok = true
			*out = tw(*out, fd)
			*er = tw(*er, fd)
			return
		}
	case "&>>":
		return func(f string, in *io.Reader, out, er *io.Writer) (ok bool) {
			fd, err := os.OpenFile(f, os.O_RDWR|os.O_APPEND, 0)
			if err != nil {
				fmt.Printf("%s: open file failed: %s\n", os.Getenv("SHELL"), f)
				return
			}
			ok = true
			*out = tw(*out, fd)
			*er = tw(*er, fd)
			return
		}
	case "1<&1", "1<&2", "2<&1", "2<&2", "1>&0", ">&0", "2>&0":
		return func(f string, in *io.Reader, out, er *io.Writer) (ok bool) {
			fmt.Printf("%s: invalid redirect: %s\n", os.Getenv("SHELL"), symbol)
			return
		}
	case "2>&1", "1>&1", ">&1", "1>&2", ">&2", "2>&2", "0<&1", "<&1", "0<&2", "<&2", "<<<", "<<":
		return func(f string, in *io.Reader, out, er *io.Writer) (ok bool) {
			// TODO
			return
		}
	default:
		return nil
	}
}

func tr(r, r1 io.Reader) io.Reader {
	if r == nil {
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

func (c ChannelRW) Write(p []byte) (n int, err error) {
	defer func() {
		if err := recover(); err != nil {
			n = 0
			err = errors.New("channel closed")
		}
	}()
	c <- p
	return len(p), nil
}

func (c ChannelRW) Read(p []byte) (n int, err error) {
	s, ok := <-c
	if ok {
		return copy(p, s), nil
	}
	return 0, io.EOF
}
