package raft

import (
	"context"
	"net/rpc"
	"sync"

	"github.com/krantius/logging"
)

func (r *Raft) appendEntries(args AppendEntriesArgs, res *AppendEntriesResponse) error {
	// Reset the election timer
	r.electionTimer.Reset(r.electionTimeout)

	r.mu.Lock()
	defer r.mu.Unlock()

	res.Term = r.term

	// If leader and request term is less, ignore
	if args.Term < r.term && r.state == Leader {
		res.Success = false
		return nil
	}

	r.election = nil
	r.candidacy = nil
	r.state = Follower

	// Heartbeat
	if args.Entries == nil {
		if args.LeaderCommit == r.log.CommitIndex {
			logging.Infof("%d = %d. skipping", args.LeaderCommit, r.log.CommitIndex)
			res.Success = true
			return nil
		}

		// Heartbeat beat the actual append stuff, TODO fix this
		if args.LeaderCommit > r.log.CurrentIndex {
			logging.Infof("Bad ordering of commit and append, leadCommit = %d curIndex = %d", args.LeaderCommit, r.log.CurrentIndex)
			res.Success = false
			return nil
		}

		// Need to commit stuff still
		logging.Debugf("Committing index %d from heartbeat", args.LeaderCommit)

		if commits := r.log.commit(args.LeaderCommit); commits != nil {
			for _, c := range commits {
				switch c.Cmd.Op {
				case Set:
					r.fsm.Set(c.Cmd.Key, c.Cmd.Val)
				case Delete:
					r.fsm.Delete(c.Cmd.Key)
				}
			}
		}

		res.Success = true
		return nil
	}

	res.Success = r.log.Append(args.Term, args.PrevLogIndex, args.PrevLogTerm, args.LeaderCommit, args.Entries)

	// Need to actually update the fsm with the entries that were committed...

	return nil
}

func (r *Raft) appendAll(lastIndex int, entries []LogEntry) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	wg := sync.WaitGroup{}

	var mu sync.Mutex
	commitCount := 1

	logging.Infof("Leader commit index = %d", r.log.CommitIndex)

	for _, peer := range r.peers {
		wg.Add(1)

		go func(peer string) {
			defer wg.Done()

			l := r.peerLogs[peer]

			args := AppendEntriesArgs{
				Term:         r.term,
				LeaderId:     r.id,
				LeaderCommit: r.log.CommitIndex,
				PrevLogIndex: l.Index,
				PrevLogTerm:  l.Term - 1,
				Entries:      entries,
			}

			res, err := r.callAppendEntries(peer, args)
			if err != nil {
				logging.Errorf("callAppendEntries failed for peer %q: %v", peer, err)
				return
			}

			if entries != nil {
				r.peerLogs[peer] = entries[len(entries)-1]
				logging.Infof("Updated peer log %s %+v", peer, r.peerLogs[peer])
			}

			if res.Success {
				mu.Lock()
				commitCount++
				mu.Unlock()
			} else {
				// TODO handle non-success case
			}
		}(peer)
	}

	wg.Wait()

	if commitCount >= (len(r.peers)+1)/2 {
		r.log.commit(lastIndex)
		return true
	}

	return false
}

func (r *Raft) callAppendEntries(addr string, args AppendEntriesArgs) (AppendEntriesResponse, error) {
	res := AppendEntriesResponse{}

	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		logging.Infof("RPC Client failed: %v\n", err)
		return res, err
	}

	if err := client.Call("Raft.AppendEntries", args, &res); err != nil {
		logging.Infof("Failed to callRequestVote: %v", err)
		return res, err
	}

	return res, nil
}

func (r *Raft) heartbeat(ctx context.Context) {
	for {
		select {
		case <-r.heartbeatTimer.C:
			r.appendAll(r.log.CommitIndex, nil)

			r.heartbeatTimer.Reset(r.heartbeatTimeout)
		case <-ctx.Done():
			logging.Infof("%s heartbeat stopping", r.id)
			r.heartbeatTimer.Stop()
			return
		}
	}
}
