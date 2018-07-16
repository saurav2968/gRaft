package core

import "github.com/sirupsen/logrus"

type behaviour interface{
	setup() 	//give it capabilities
	operate()   //start a for loop listening for messages using select
	tearDown()  // cleanup if any
}

/***** DORMANT FOLLOWER *******/

type dormantFollower struct{
	parent *consensusServer
	capabilities []capabilities
}

func (d dormantFollower) setup(){
	logrus.Infof("Setting up Dormant Follower.")
	logrus.Infof("Capabilities: %v", d.capabilities)

}

/*
	It gets messages from gateway/api via channel.
	Also reads cfg and does all capabilities task if any,
	adds new servers
 */
func (d dormantFollower) operate(){
	logrus.Infof("Dormant Follower ready to operate")
	for{
		select {
		case rawMsg := <-d.parent.tcpServerToConsensus:
			switch msg := rawMsg.(type) {
			case addServerMsg:
				logrus.Infof("Received addserver msg from tcpserver: %s", msg)
				// create a gateway, send it a hdnashake msg with our cluster view and now check for quorum met
				to := make(chan interface{}, 100)
				from := make(chan interface{}, 100)
				g := gateway{
					conn: msg.conn,
					fromConsensus: from,
					toConsensus: to,
				}
				d.parent.connToGateway[msg.conn] = g
				g.toConsensus <- handshake{address{}, false, getClusterAddress(d.parent.connToGateway)}
			}
		}
	}
}

func (d dormantFollower) tearDown(){
	logrus.Infof("Tearing down Dormant Follower")
}

/***** TIMER FOLLOWER *******/