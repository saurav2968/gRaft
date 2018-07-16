package core

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"strings"
	"strconv"
)

func checkPort(port uint) {
	logrus.Infof("Checking port: %d", port)
	l, err := net.Listen("tcp", ":"+fmt.Sprint(port))

	if err != nil {
		logrus.Fatalf("Unable to listen on port: %s\nError: %s", port, err)
	}
	l.Close()
}

func writable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}

func checkAndCreateStateDir(stateDir string) {
	logrus.Infof("Checking State Dir: %s", stateDir)

	if _, err := os.Stat(stateDir); err != nil {
		if os.IsNotExist(err) {
			//create dir
			if err := os.MkdirAll(stateDir, RAFT_STATE_DIR_PERMS); err != nil {
				logrus.Fatalf("Unable to create directory: %s for Raft state\nErr: %s", stateDir, err)
			}
			logrus.Infof("Created Raft state directory: %s", stateDir)
		} else {
			logrus.Fatalf("Error while checking for Raft State directory:\n%s", err)
		}
	}
	if writable(stateDir) == false {
		logrus.Fatalf("Raft state directory: %s is not writable!", stateDir)
	}
}

func ValidateConfig(r RaftConfig) {
	//check port available
	checkPort(r.GatewayPort)
	checkPort(r.ApiPort)

	//check state dir available and is writable
	checkAndCreateStateDir(r.StateDir)
}


func getClusterAddress(connToGateway map[net.Conn]gateway) []address{
	addresses := []address{}
	for c, _ := range connToGateway{
		remoteAddStr := c.RemoteAddr().String()
		logrus.Debugf("Remote address: %v", remoteAddStr)
		a := strings.Split(remoteAddStr, ":")
		port, _ := strconv.ParseUint(a[1], 10, 32) //returns uint64
		addresses = append(addresses, address{a[0], uint(port)})
	}

}