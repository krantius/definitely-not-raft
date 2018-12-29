package raft

import (
	"fmt"
	"net"

	"github.com/krantius/definitely-not-raft/shared/logging"
)

func (n *Node) listen() {
	logging.Infof("%s listening %d", n.id, n.port)
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", n.port))
	if err != nil {
		logging.Errorf("listen error: %v", err)
		panic(err)
	}

	n.server.Accept(l)

	logging.Info("listen ending")
}
