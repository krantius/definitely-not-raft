package raft

import (
	"context"
	"math/rand"
	"net/http"
	"net/rpc"
	"sync"
	"time"

	"github.com/krantius/definitely-not-raft/shared/logging"
)

// Node is a raft node in a cluster
type Node struct {
	id    string
	port  int
	term  int
	state State

	// election stuff
	election  *Election
	candidacy *Candidacy

	// timers
	electionTimer   *time.Timer
	electionTimeout time.Duration

	heartbeatTimer   *time.Timer
	heartbeatTimeout time.Duration
	heartbeatCancel  context.CancelFunc

	// concurrency
	mu sync.Mutex

	// Connection Stuff
	server *rpc.Server
	peers  []string
}

func NewNode(id string, port int, peers []string) *Node {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	timeout := time.Duration(r1.Intn(10)+5) * time.Second
	hbTimeout := 3 * time.Second

	logging.Infof("%s %v", id, timeout)

	node := &Node{
		id:               id,
		port:             port,
		peers:            peers,
		state:            Follower,
		electionTimer:    time.NewTimer(timeout),
		electionTimeout:  timeout,
		heartbeatTimer:   time.NewTimer(hbTimeout),
		heartbeatTimeout: hbTimeout,
	}

	node.server = rpc.NewServer()

	oldMux := http.DefaultServeMux
	mux := http.NewServeMux()
	http.DefaultServeMux = mux

	node.server.RegisterName("Raft", node)
	node.server.HandleHTTP("/", "/debug")

	http.DefaultServeMux = oldMux

	return node
}

func (n *Node) Do() {
	go n.electionCountdown()
	n.listen()

	logging.Infof("%s exiting", n.id)
}
