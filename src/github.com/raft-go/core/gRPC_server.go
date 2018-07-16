package core

type gRPCServer struct {
	r *raft
	port uint
	toConsensus chan interface{}
}


func (g *gRPCServer) start(){

}