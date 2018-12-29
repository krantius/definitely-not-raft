package raft

import (
	"context"
	"net/rpc"
	"sync"

	"github.com/krantius/definitely-not-raft/shared/logging"
	log "github.com/sirupsen/logrus"
)

func (n *Node) RequestVote(args RequestVoteArgs, res *RequestVoteResponse) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state == Candidate {
		if args.Term > n.candidacy.term {
			res.VoteGranted = true

			n.election = &Election{
				term:  args.Term,
				voted: args.CadidateId,
			}
		}
		res.VoteGranted = false

		return nil
	}

	if n.election != nil {
		// Already voted, skip
		res.VoteGranted = false
		return nil
	}

	if args.Term <= n.term {
		// Not a new term, continue with our election
		res.VoteGranted = false
		return nil
	}

	n.election = &Election{
		term:  args.Term,
		voted: args.CadidateId,
	}

	if n.state == Leader {
		if n.heartbeatCancel != nil {
			n.heartbeatCancel()
			n.heartbeatCancel = nil
		}
	}

	logging.Infof("%s voting for %s", n.id, args.CadidateId)
	res.VoteGranted = true

	return nil
}

func (n *Node) callRequestVote(addr string) bool {
	log.Infof("Node %s calling for a vote from %s", n.id, addr)

	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		log.Infof("RPC Client failed: %v\n", err)
		return false
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	args := RequestVoteArgs{
		Term:       n.term,
		CadidateId: n.id,
	}

	res := &RequestVoteResponse{}

	if err := client.Call("Raft.RequestVote", args, res); err != nil {
		log.Infof("Failed to callRequestVote: %v", err)
		return false
	}

	return res.VoteGranted
}

func (n *Node) electionCountdown() {
	for {
		select {
		case <-n.electionTimer.C:
			n.mu.Lock()

			if n.state == Leader {
				log.Warnf("%s tried to start election when leader", n.id)
				break
			}

			n.electionTimer.Reset(n.electionTimeout)

			logging.Infof("%s starting election", n.id)

			n.term++

			n.candidacy = &Candidacy{
				votes: 1,
				term:  n.term,
			}

			n.state = Candidate

			n.mu.Unlock()

			// Do vote requests
			wg := sync.WaitGroup{}

			for _, peer := range n.peers {
				wg.Add(1)
				go func(p string) {
					defer wg.Done()

					if n.callRequestVote(p) {
						n.mu.Lock()
						n.candidacy.votes++
						n.mu.Unlock()
					}

				}(peer)
			}

			wg.Wait()

			if n.candidacy.votes > (len(n.peers)+1)/2 {
				n.state = Leader
				n.electionTimer.Stop()

				var ctx context.Context
				ctx, n.heartbeatCancel = context.WithCancel(context.Background())

				go n.heartbeat(ctx)
			}
		}
	}
}
