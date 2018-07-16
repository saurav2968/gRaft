package core

import (
	"github.com/sirupsen/logrus"
	"time"
)

type raft struct {
	config RaftConfig
	mode   RaftMode
}

func (r *raft) startGuardian() {
	logrus.Infof("*************************************************************")
	logrus.Infof("Starting RAFT Guardian...")

	numConsensusRestarts := 0
	numgRPCServerRestarts := 0

	consensusSignal := make(chan bool)
	consensusMsg := make(chan string)
	apiToConsensus := make(chan interface{}, 100)

	gRPCServerSignal := make(chan bool)
	gRPCServerMsg := make(chan string)
	gRPCServerToConsensus := make(chan interface{}, 100)

	// This guy should start all goroutines unders supervisor eventually
	go r.startConsensusModule(consensusMsg, consensusSignal, apiToConsensus, gRPCServerToConsensus)
	go r.startgRPCServer(r.config.GatewayPort, gRPCServerMsg, gRPCServerSignal, gRPCServerToConsensus)
	for {
		select {
		case msg := <-consensusMsg:
			switch msg {
			case BOOTSTRAPPED:
				logrus.Infof("Consensus module ready...")
			}
		case exitStatus := <-consensusSignal:
			numConsensusRestarts++
			logrus.Infof("consensus module killed with status: %v", exitStatus)
			if exitStatus == false {
				logrus.Infof("Restart consensus module: attempt %d", numConsensusRestarts)
				go r.startConsensusModule(consensusMsg, consensusSignal, apiToConsensus, gRPCServerToConsensus)
			}
			// We dont have default case for now as we dont want to do anything other than waiting
			/*default:
			fmt.Println("no activity")*/
		case msg := <-gRPCServerMsg:
			switch msg {
			case BOOTSTRAPPED:
				logrus.Infof("gRPC Server module ready...")
			}
		case exitStatus := <-gRPCServerSignal:
			numConsensusRestarts++
			logrus.Infof("consensus module killed with status: %v", exitStatus)
			if exitStatus == false {
				logrus.Infof("Restart consensus module: attempt %d", numConsensusRestarts)
				go r.startgRPCServer(r.config.GatewayPort, gRPCServerMsg, gRPCServerSignal, gRPCServerToConsensus)
			}
			// We dont have default case for now as we dont want to do anything other than waiting
			/*default:
			fmt.Println("no activity")*/
		}
	}
}

/*func (r *raft) Boostrapped bool(){

}*/
/********* CORE MODULES **********/
//started by guradian actor as goroutines

func (r *raft) startConsensusModule(msgChan chan string, signalChan chan bool, apiToConsensus chan interface{},
	gRPCServerToConsensus chan interface{}) {
	exitStatus := false
	defer func() {
		signalChan <- exitStatus
		logrus.Infof("Consensus Module done.")
	}()

	logrus.Infof("Starting Consensus module...")
	msgChan <- BOOTSTRAPPED

	c := consensusServer{
		r: r,
		apiToConsensus: apiToConsensus,
		gRPCServerToConsensus: gRPCServerToConsensus,
	}

	c.start()

	time.Sleep(30 * time.Second)
}

func (r *raft) startgRPCServer(port uint, msgChan chan string, signalChan chan bool, gRPCServerToConsensus chan interface{}) {
	exitStatus := false
	defer func() {
		signalChan <- exitStatus
		logrus.Infof("gRPC Server done.")
	}()

	logrus.Infof("Starting gRPC Server...")
	msgChan <- BOOTSTRAPPED

	t := gRPCServer{
		r: r,
		port: port,
		toConsensus: gRPCServerToConsensus,
	}

	t.start()
}
