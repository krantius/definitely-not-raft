package raft

import (
	"context"
	"net/rpc"
	"sync"

	"github.com/krantius/logging"
)

func (r *Raft) requestVote(args RequestVoteArgs, res *RequestVoteResponse) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == Candidate {
		if args.Term > r.candidacy.term {
			res.VoteGranted = true

			r.election = &Election{
				term:  args.Term,
				voted: args.CadidateId,
			}
		}
		res.VoteGranted = false

		return nil
	}

	if r.election != nil {
		// Already voted, skip
		res.VoteGranted = false
		return nil
	}

	if args.Term <= r.term {
		// Not a new term, continue with our election
		res.VoteGranted = false
		return nil
	}

	r.election = &Election{
		term:  args.Term,
		voted: args.CadidateId,
	}

	if r.state == Leader {
		if r.heartbeatCancel != nil {
			r.heartbeatCancel()
			r.heartbeatCancel = nil
		}
	}

	r.term = args.Term

	logging.Infof("%s voting for %s", r.id, args.CadidateId)
	res.VoteGranted = true

	return nil
}

func (r *Raft) callRequestVote(addr string) bool {
	logging.Infof("Node %s calling for a vote from %s", r.id, addr)

	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		logging.Infof("RPC Client failed: %v\n", err)
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	args := RequestVoteArgs{
		Term:       r.term,
		CadidateId: r.id,
	}

	res := &RequestVoteResponse{}

	if err := client.Call("Raft.RequestVote", args, res); err != nil {
		logging.Infof("Failed to callRequestVote: %v", err)
		return false
	}

	return res.VoteGranted
}

func (r *Raft) doElection() {
	logging.Info("Starting election")

	// Do vote requests
	wg := sync.WaitGroup{}

	for _, peer := range r.peers {
		wg.Add(1)

		go func(p string) {
			defer wg.Done()

			if r.callRequestVote(p) {
				r.mu.Lock()
				r.candidacy.votes++
				r.mu.Unlock()
			}

		}(peer)
	}

	wg.Wait()

	if r.candidacy.votes >= (len(r.peers)+1)/2 {
		r.state = Leader
		r.electionTimer.Stop()

		var ctx context.Context
		ctx, r.heartbeatCancel = context.WithCancel(context.Background())

		go r.heartbeat(ctx)
	}
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

			r.candidacy = &Candidacy{
				votes: 1,
				term:  r.term,
			}

			r.state = Candidate
			r.mu.Unlock()

			r.doElection()
		}
	}
}
