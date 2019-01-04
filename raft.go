package raft

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/krantius/logging"
)

// Raft is a raft node in a cluster
type Raft struct {
	// Config stuff
	id    string
	port  int
	term  int
	state State

	// Election stuff
	election  *Election
	candidacy *Candidacy

	// Timers
	electionTimer   *time.Timer
	electionTimeout time.Duration

	heartbeatTimer   *time.Timer
	heartbeatTimeout time.Duration
	heartbeatCancel  context.CancelFunc

	// Concurrency
	mu sync.Mutex

	// Connection Stuff
	rpc   *rpcServer
	peers []string

	// Data handling
	fsm      Store
	log      *Log
	peerLogs map[string]LogEntry
}

func New(cfg Config, fsm Store) *Raft {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	timeout := time.Duration(r1.Intn(10000)+500) * time.Millisecond
	hbTimeout := 3 * time.Second

	logging.Infof("Raft starting with electrion timeout %v", timeout)

	ra := &Raft{
		id:               cfg.ID,
		port:             cfg.Port,
		peers:            cfg.Peers,
		fsm:              fsm,
		state:            Follower,
		rpc:              &rpcServer{},
		electionTimer:    time.NewTimer(timeout),
		electionTimeout:  timeout,
		heartbeatTimer:   time.NewTimer(hbTimeout),
		heartbeatTimeout: hbTimeout,
		log:              &Log{},
	}

	ra.rpc.requestCb = ra.requestVote
	ra.rpc.appendCb = ra.appendEntries

	ra.peerLogs = make(map[string]LogEntry, len(cfg.Peers))
	for _, val := range cfg.Peers {
		ra.peerLogs[val] = LogEntry{}
	}

	return ra
}

func (r *Raft) Start() {
	go r.electionCountdown()
	r.rpc.listen(r.port)

	logging.Infof("%s exiting", r.id)
}

// Apply distributes the command to the other raft nodes
//
// Hotpath used by client when making state changes
func (r *Raft) Apply(c Command) error {
	if r.state != Leader {
		// TODO forward to leader?
		return errors.New("not leader")
	}

	// Append to local log
	log := r.log.appendCmd(c)

	// Call AppendEntries to peers
	committed := r.appendAll(log.Index, []LogEntry{log})

	// Once we get a quorem from the other nodes, commit and callback to client to update the real map
	// or do this somewhere else in another callback...

	if !committed {
		return nil
	}

	switch c.Op {
	case Set:
		r.fsm.Set(c.Key, c.Val)
	case Delete:
		r.fsm.Delete(c.Key)
	}

	return nil
}
