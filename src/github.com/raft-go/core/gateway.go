package core

import "net"

type gateway struct {
	conn net.Conn
	fromConsensus chan interface{}
	toConsensus chan interface{}
}
