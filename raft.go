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
	peerLogs map[string]*peer
}

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
	for _, val := range cfg.Peers {
		ra.peerLogs[val] = &peer{
			addr:  val,
			l:     ra.log,
			state: stateSynced,
			current: LogEntry{
				Index: -1,
				Term:  0,
			},
			ctx: context.Background(), // TODO  not use background
		}
	}

	return ra
}

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

func (r *Raft) Dump() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	str := fmt.Sprintf("id=%s state=%s term=%d logTerm=%d index=%d commitIndex=%d\n%+v", r.id, r.state, r.term, r.log.CurrentTerm, r.log.CurrentIndex, r.log.CommitIndex, r.log.logs)
	return []byte(str)
}
