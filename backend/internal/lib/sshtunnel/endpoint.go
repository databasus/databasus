package sshtunnel

import (
	"net"
	"strconv"
)

type Endpoint struct {
	Host string
	Port int
}

func (e Endpoint) String() string {
	return net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
}
