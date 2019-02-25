package raft

import (
	"context"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/krantius/logging"
)

const (
	voteRpcTimeout time.Duration = 500 * time.Millisecond
)

// Election
type Election struct {
	term  int
	voted string
}

type Candidacy struct {
	term  int
	votes int
}

type State string

const (
	Follower  State = "follower"
	Candidate State = "candidate"
	Leader    State = "leader"
)

func (r *Raft) requestVote(args RequestVoteArgs, res *RequestVoteResponse) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == Candidate {
		if args.Term > r.term {
			logging.Infof("Candidate voting for %s", args.CadidateID)
			res.VoteGranted = true

			r.election = &Election{
				term:  args.Term,
				voted: args.CadidateID,
			}

			r.state = Follower
		} else {
			res.VoteGranted = false
		}

		return nil
	}

	if r.election != nil {
		// Already voted, skip
		res.VoteGranted = false
		return nil
	}

	if args.Term <= r.term {
		// Not a new term
		res.VoteGranted = false
		return nil
	}

	r.election = &Election{
		term:  args.Term,
		voted: args.CadidateID,
	}

	if r.state == Leader {
		r.stepDown()
	}

	r.term = args.Term

	logging.Infof("Voting for %s", args.CadidateID)
	res.VoteGranted = true

	return nil
}

func (r *Raft) callRequestVote(addr string) bool {
	logging.Infof("Node %s calling for a vote from %s", r.id, addr)

	conn, err := net.DialTimeout("tcp", addr, voteRpcTimeout)
	if err != nil {
		logging.Infof("RPC Client dial failed: %v\n", err)
		return false
	}

	client := rpc.NewClient(conn)

	r.mu.Lock()
	defer r.mu.Unlock()

	args := RequestVoteArgs{
		Term:       r.term,
		CadidateID: r.id,
	}

	res := &RequestVoteResponse{}

	if err := client.Call("Raft.RequestVote", args, res); err != nil {
		logging.Infof("Failed to callRequestVote: %v", err)
		return false
	}

	return res.VoteGranted
}

func (r *Raft) callElection() {
	logging.Info("Starting election")

	// Do vote requests
	wg := sync.WaitGroup{}

	c := Candidacy{
		term: r.term,
	}

	var mu sync.Mutex

	for _, peer := range r.peers {
		wg.Add(1)

		go func(p string) {
			defer wg.Done()

			if r.callRequestVote(p) {
				mu.Lock()
				c.votes++
				mu.Unlock()
			}

		}(peer)
	}

	wg.Wait()

	if c.votes >= (len(r.peers)+1)/2 {
		r.stepUp()
	} else {
		// Lost election
		r.state = Follower
	}
}

func (r *Raft) stepUp() {
	// Step up as leader...
	logging.Info("Becoming leader...")
	r.state = Leader
	r.electionTimer.Stop()

	var ctx context.Context
	ctx, r.heartbeatCancel = context.WithCancel(context.Background())

	for i, val := range r.peers {
		cancelCtx, cancel := context.WithCancel(context.Background())
		r.peerLogs[val] = &peer{
			addr:  val,
			l:     r.log,
			state: stateSynced,
			current: LogEntry{
				Index: r.log.CurrentIndex,
				Term:  r.log.CurrentTerm,
			},
			ctx: cancelCtx,
		}

		r.peerCancels[i] = cancel
	}

	r.log.CurrentTerm = r.term

	go r.heartbeat(ctx)
}

func (r *Raft) stepDown() {
	// Step down from leader...
	if r.heartbeatCancel != nil {
		r.heartbeatCancel()
		r.heartbeatCancel = nil
	}

	for _, cancel := range r.peerCancels {
		if cancel != nil {
			cancel()
		}
	}

	r.state = Follower
}

func (r *Raft) electionCountdown() {
	for {
		select {
		case <-r.electionTimer.C:
			r.mu.Lock()

			if r.state == Leader {
				logging.Warningf("%s tried to start election when leader", r.id)
				break
			}

			r.electionTimer.Reset(r.electionTimeout)

			r.term++
			r.state = Candidate
			r.mu.Unlock()

			r.callElection()
		}
	}
}
