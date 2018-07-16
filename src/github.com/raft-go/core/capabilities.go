package core

type capabilities uint

const (
	TIME capabilities = iota
	HEARTBEAT
	CLIENTREAD
	CLIENTWRITE
	DISCOVERCLUSTER
	ACCEPTNEWSERVER
	REMOVESERVER
	FINDCLUSTER //read COMMITED cluster cfg and checks for diff between gateways and cfg and connects/disconnects servers
)