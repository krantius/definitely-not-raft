package raft

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"

	"github.com/krantius/logging"
)

type rpcServer struct {
	server    *rpc.Server
	appendCb  func(args AppendEntriesArgs, res *AppendEntriesResponse) error
	requestCb func(args RequestVoteArgs, res *RequestVoteResponse) error
}

func (r *rpcServer) AppendEntries(args AppendEntriesArgs, res *AppendEntriesResponse) error {
	return r.appendCb(args, res)
}

func (r *rpcServer) RequestVote(args RequestVoteArgs, res *RequestVoteResponse) error {
	return r.requestCb(args, res)
}

func (r *rpcServer) listen(port int) {
	r.server = rpc.NewServer()

	oldMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()

	r.server.RegisterName("Raft", r)
	r.server.HandleHTTP("/", "/debug")

	http.DefaultServeMux = oldMux

	logging.Infof("Listening on port %d...", port)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logging.Errorf("Listen error: %v", err)
		panic(err)
	}

	r.server.Accept(l)

	logging.Debugf("Listen ending")
}
