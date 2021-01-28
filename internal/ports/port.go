package ports

import (
	"github.com/jschaf/pggen/internal/errs"
	"net"
)

// Port is a port.
type Port = int

// FindAvailable returns an available port by asking the kernel for an
// unused port. https://unix.stackexchange.com/a/180500/179300
//
// Copied and slightly modified from https://github.com/phayes/freeport
// Licensed under BSD-3. Copyright (c) 2014, Patrick Hayes
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
