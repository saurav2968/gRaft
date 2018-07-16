package core

import "net"

type address struct {
	host string
	port uint
}
/***** Consensus to Guradian ******/
const BOOTSTRAPPED = "bootstrapped"

type addServerMsg struct {
	conn net.Conn
}

type handshake struct {
	leaderAddress address
	isLeader bool
	clusterNodes []address
}

/****** cluster messages *********/

/*
	1. AppendEntry
	2. RequestVote
	3. Install
	4. ClusterJoin/Handshake
	5. Induct
	6. Bootstrap(leader -> inductee)
	7. Bootstrapped
	8. Inducted
	9. InstallIndex

	Note: To add a member to cluster, you go via REST API, The recipient gets to know the host and port of joinee,
	and in response sends all cluster entities OR a 302 with leader address.
	The caller joins via an RPC and in response gets cluster view and/OR leader address.
	So 2 i/f here.One via REST and one via RPC.
*/
