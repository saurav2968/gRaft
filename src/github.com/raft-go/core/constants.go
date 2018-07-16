package core

const (
	RAFT_SINGLE_MODE RaftMode = iota
	RAFT_CLUSTER_MODE
)

const RAFT_STATE_DIR_PERMS = 0700
