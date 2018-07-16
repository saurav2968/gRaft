package core

import (
	"github.com/sirupsen/logrus"
	"net"
)

type ServerRole uint
type ServerState uint

func (s ServerRole) String() string{
	switch s{
	case LEADER:
		return "LEADER"
	case FOLLOWER:
		return "FOLLOWER"
	case CANDIDATE:
		return "CANDIDATE"
	}
	return "unknown"
}

func (s ServerState) String() string{
	switch s {
	case DORMANT:
		return "DORMANT"
	case TIMER:
		return "TIMER"
	case INDUCTEE:
		return "INDUCTEE"
	case BOOTSTRAP:
		return "BOOTSTRAP"
	case OPERATIONAL:
		return "OPERATIONAL"
	}
	return "unknown"
}

const (
	LEADER ServerRole = iota
	FOLLOWER
	CANDIDATE
)

/*
	------ Description of serverstate in FOLLOWER MODE --------

	All followers start in DORMANT mode in which it just wants to be part of cluster.
	DORMANT
	_______
	It reads cluster cfg if any and tries to establish connection with nodes.
		1.0 If no nodes, it means this is a fresh node and hence needs to be added to a cluster OR some other node needs to be added here.
			1.1 Either add this to another node:
				The other node may be in any role/mode.
				If there is a leader, it forwards leader address and this node then forwards to leader.
				Else this node forwards its view of cluster to us.
				We should then repeat the same with other nodes.
				With every new connection we check if quorum met.If yes we switch to TIMER mode.
			1.2 We Get a Request to add node:
				Accept connection and send our view as handshake message.
				Now if quorum met, switch to TIMER mode

		2.0 If we find nodes, we start the same procedure as 1.1. We still need to service new node addition requests.
	In DORMANT mode we still need to reply to RequestVote/AppendRPC. On RequestVote we just vote.
	On AppendRPC, we switch to INDCUTEE mode

	TIMER
	-----
	In this case it still answers to new node addition request.
	Quorum has been fulfilled and it sets up timers.
	Now it checks if node disconnects and switches to DORMANT mode.
	If before its timer expires, if it gets RequestVote it votes,
	If it gets AppendRPC, it accepts and switches to INDCUTEE.

	INDUCTEE
	--------
	In this state, it needs to send an INDUCTRPC to leader asking it to bootstrap it.
	Then it swicthes to BOOTSTRAP MODE

	BOOTSTRAP
	---------
	It then checks if it has local snapshot and if yes, bootstrap from it.
	It gets AppendRPC in this case but replies with same index OR
	It get only heartbeats.Once it has restored from snapshot, it sends FOLLOWERBOOTSTRAPPEDRPC.
	Leader now starts sending backlog(via regular AppendRPCs) and once folower has caught up , it updates its write status
	and sends INDUCTEDRPC upon which switches to OPERATIONAL state.

	OPERATIONAL
	-----------
	Now it does normal stuff.

	In all state, if connection is lost, then every node evaluates its quorum and may switch to DORMANT state.
	In this state if it gets to know a candidate(RequestVote) OR leader(AppendRPC), then it should ask for their view
	too.So all AppendRPC/RequestVote sends view of cluster. And then it follows 1.1 step and finally acks the message.

	------ Description of serverstate in LEADER MODE --------

	When a node becomes LEADER, it goes into BOOTSTRAP mode.
	Here it issues heartbeat,installrpc etc. But doesnt service client requests.
	It does a dummy commit and this way it tries to find commitIndex(max(log entry on quorum nodes)).
	Once commitIndex found, it goes to Operational state and takes client requests.
*/

/*
	Every serverstate is a behaviour which is an implementation of an interface, which specifies a setup phase
	giving it 'capabilities'. These capabilities come from server's role+ state.
	And then a decay phase when it transitions to different serverstate + role.
 */
const (
	DORMANT ServerState = iota
	TIMER
	INDUCTEE
	BOOTSTRAP
	OPERATIONAL
)

type consensusServer struct {
	r     *raft
	role  ServerRole
	state ServerState
	apiToConsensus chan interface{} //maybe change it to struct
	tcpServerToConsensus chan interface{} //maybe change it to struct
	connToGateway map[net.Conn]gateway
}

func (c *consensusServer) start() {
	if c.r.mode == RAFT_CLUSTER_MODE {
		//start an fsm with follower role and dormant state
		c.role = FOLLOWER
		c.state = DORMANT
		c.setupBehaviour()
	}
}

func (c *consensusServer) setupBehaviour(){
	logrus.Infof("Setting up server with state: %s/%s",c.role, c.state)
	d := c.newFollower()
	c.cycle(d)
}

func (c *consensusServer) newFollower() behaviour{
	if c.role == FOLLOWER{
		if c.state == DORMANT{
			return dormantFollower{
				parent: c,
				capabilities: []capabilities{
					ACCEPTNEWSERVER,
					},
			}
		}
	}
	return nil
}

func (c *consensusServer) cycle(t behaviour){
	t.setup()
	t.operate() //sets up next behaviour using parent pointer
	t.tearDown()
	c.setupBehaviour()
}
