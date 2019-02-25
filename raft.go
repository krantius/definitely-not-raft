package raft

import (
	"context"
	"errors"
	"fmt"
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
	ctx   context.Context

	// Election stuff
	election *Election

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
	fsm Store
	log *Log

	// Peers
	peerLogs    map[string]*peer
	peerCancels []context.CancelFunc
}

// Config contains the settings needed to start a raft node
type Config struct {
	ID    string
	Port  int
	Peers []string
}

// New creates a new raft node
func New(ctx context.Context, cfg Config, fsm Store) *Raft {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	timeout := time.Duration(r1.Intn(10000)+5000) * time.Millisecond
	hbTimeout := 1 * time.Second

	logging.Infof("Raft starting with election timeout %v", timeout)

	ra := &Raft{
		ctx:              ctx,
		id:               cfg.ID,
		port:             cfg.Port,
		peers:            cfg.Peers,
		fsm:              fsm,
		state:            Follower,
		rpc:              &rpcServer{},
		electionTimer:    time.NewTimer(timeout),
		electionTimeout:  timeout,
		heartbeatTimer:   time.NewTimer(0),
		heartbeatTimeout: hbTimeout,
		log: &Log{
			CommitIndex:  -1,
			CurrentIndex: -1,
			fsm:          fsm,
		},
	}

	ra.rpc.requestCb = ra.requestVote
	ra.rpc.appendCb = ra.appendEntries

	ra.peerLogs = make(map[string]*peer, len(cfg.Peers))
	ra.peerCancels = make([]context.CancelFunc, len(cfg.Peers))

	return ra
}

// Start begins the election countdown and starts the rpc server
func (r *Raft) Start() {
	go r.electionCountdown()
	go r.rpc.listen(r.port)

	<-r.ctx.Done()

	logging.Info("Raft exiting")
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
	r.appendAll([]LogEntry{log})

	return nil
}

// Dump is currently used to get the state of the raft server for debugging
// TODO delete this
func (r *Raft) Dump() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	str := fmt.Sprintf("id=%s state=%s term=%d logTerm=%d index=%d commitIndex=%d\n%+v", r.id, r.state, r.term, r.log.CurrentTerm, r.log.CurrentIndex, r.log.CommitIndex, r.log.logs)
	return []byte(str)
}
