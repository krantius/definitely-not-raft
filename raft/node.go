package raft

import (
	"context"
	"math/rand"
	"net/http"
	"net/rpc"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/krantius/definitely-not-raft/raft/replication"
	"github.com/krantius/definitely-not-raft/shared/logging"
)

// Node is a raft node in a cluster
type Node struct {
	// Config stuff
	id    string
	port  int
	term  int
	state State

	// Election stuff
	election  *Election
	candidacy *Candidacy

	// Log stuff
	log *replication.Log

	// Timers
	electionTimer   *time.Timer
	electionTimeout time.Duration

	heartbeatTimer   *time.Timer
	heartbeatTimeout time.Duration
	heartbeatCancel  context.CancelFunc

	// Concurrency
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
	http.DefaultServeMux = http.NewServeMux()

	node.server.RegisterName("Raft", node)
	node.server.HandleHTTP("/", "/debug")

	http.DefaultServeMux = oldMux

	r := mux.NewRouter()
	sr := r.PathPrefix("/api").Subrouter()
	sr.Path("/status").Methods("GET").HandlerFunc(node.Status)

	go http.ListenAndServe(":8000", r)

	return node
}

func (n *Node) Do() {
	go n.electionCountdown()
	n.listen()

	logging.Infof("%s exiting", n.id)
}
