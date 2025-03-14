package ports

import (
	"net"

	"github.com/jschaf/pggen/internal/errs"
)

// Port is a port.
type Port = int

// Licensed under BSD-3. Copyright (c) 2014, Patrick Hayes.
func FindAvailable() (p Port, mErr error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer errs.Capture(&mErr, l.Close, "")
	return l.Addr().(*net.TCPAddr).Port, nil
}
