package raft

import (
	"context"
	"net/rpc"
	"sync"

	"github.com/krantius/logging"
)

func (r *Raft) appendEntries(args AppendEntriesArgs, res *AppendEntriesResponse) error {
	logging.Debugf("Got AppendEntry from %s", args.LeaderId)
	// Reset the election timer
	r.electionTimer.Reset(r.electionTimeout)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.election = nil
	r.candidacy = nil
	r.state = Follower

	return nil
}

func (r *Raft) appendAll() error {
	wg := sync.WaitGroup{}

	for _, peer := range r.peers {
		wg.Add(1)

		go func(p string) {
			defer wg.Done()

			if err := r.callAppendEntries(p); err != nil {
				logging.Errorf("callAppendEntries failed for peer %q: %v", p, err)
			}
		}(peer)
	}

	wg.Wait()

	return nil
}

func (r *Raft) callAppendEntries(addr string) error {
	logging.Debugf("Calling AppendEntry to %s", addr)
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		logging.Infof("RPC Client failed: %v\n", err)
		return err
	}

	args := AppendEntriesArgs{
		Term:     r.term,
		LeaderId: r.id,
	}

	res := &AppendEntriesResponse{}

	if err := client.Call("Raft.AppendEntries", args, res); err != nil {
		logging.Infof("Failed to callRequestVote: %v", err)
		return err
	}

	return nil
}

func (r *Raft) heartbeat(ctx context.Context) {
	for {
		select {
		case <-r.heartbeatTimer.C:
			logging.Tracef("%s doing heartbeat", r.id)
			if err := r.appendAll(); err != nil {
				panic(err)
			}

			r.heartbeatTimer.Reset(r.heartbeatTimeout)
		case <-ctx.Done():
			logging.Infof("%s heartbeat stopping", r.id)
			r.heartbeatTimer.Stop()
			return
		}
	}
}
