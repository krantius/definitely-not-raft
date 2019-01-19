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
	appendTimeout time.Duration = 200 * time.Millisecond
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
	r.state = Follower
	r.term = args.Term

	// Heartbeat
	if args.Entries == nil {
		if args.LeaderCommit == r.log.CommitIndex {
			logging.Tracef("Commit index %d = %d. skipping", args.LeaderCommit, r.log.CommitIndex)
			res.Success = true
			return nil
		}

		// Heartbeat beat the actual append stuff, TODO fix this
		// OR this means a node is super behind and needs to be resynced
		if args.LeaderCommit > r.log.CurrentIndex {
			logging.Warningf("Follower is behind leader and needs to be resynced, leadCommit = %d curIndex = %d", args.LeaderCommit, r.log.CurrentIndex)
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

	return nil
}

func (r *Raft) appendSingle(peer string, entries []LogEntry) error {
	l := r.peerLogs[peer]

	args := AppendEntriesArgs{
		Term:         r.term,
		LeaderId:     r.id,
		LeaderCommit: r.log.CommitIndex,
		PrevLogIndex: l.Index,
		PrevLogTerm:  l.Term,
		Entries:      entries,
	}

	logging.Tracef("appendSingle peer=%s args=%+v", peer, args)

	res, err := r.callAppendEntries(peer, args)
	if err != nil {
		logging.Errorf("callAppendEntries failed for peer %q: %v", peer, err)
		return err
	}

	if res.Success {
		r.peerLogs[peer] = entries[len(entries)-1]
		logging.Debugf("Updated peer log %s %+v", peer, r.peerLogs[peer])
	} else {
		// TODO handle non-success case
		logging.Error("Single append failed D:")
		prev := r.log.walk(l)
		r.peerLogs[peer] = prev

		if err := r.appendSingle(peer, r.log.logs[l.Index:]); err != nil {
			logging.Errorf("Failed to append entries on single retry: %v", err)
		}
	}

	return nil
}

// appendAll sends out log entries to all the peers
// Leader fanning out to peers
func (r *Raft) appendAll(lastIndex int, entries []LogEntry) bool {
	//r.mu.Lock()
	//defer r.mu.Unlock()

	wg := sync.WaitGroup{}

	var mu sync.Mutex
	commitCount := 1

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
				PrevLogTerm:  l.Term,
				Entries:      entries,
			}

			logging.Tracef("appendAll peer=%s args=%+v", peer, args)

			res, err := r.callAppendEntries(peer, args)
			if err != nil {
				logging.Errorf("callAppendEntries failed for peer %q: %v", peer, err)
				return
			}

			if res.Success {
				mu.Lock()
				commitCount++
				mu.Unlock()

				if entries != nil {
					r.peerLogs[peer] = entries[len(entries)-1]
					logging.Debugf("Updated peer log %s %+v", peer, r.peerLogs[peer])
				}
			} else {
				// TODO handle non-success case
				prev := r.log.walk(l)
				r.peerLogs[peer] = prev

				if err := r.appendSingle(peer, r.log.logs[l.Index:]); err != nil {
					logging.Errorf("Failed to append entries on single retry: %v", err)
					return
				}
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

	conn, err := net.DialTimeout("tcp", addr, appendTimeout)
	if err != nil {
		logging.Infof("RPC Client dial failed: %v\n", err)
		return res, err
	}

	client := rpc.NewClient(conn)

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
			r.election = nil

			r.appendAll(r.log.CommitIndex, nil)
			r.heartbeatTimer.Reset(r.heartbeatTimeout)
		case <-ctx.Done():
			logging.Info("Heartbeat stopping")
			r.heartbeatTimer.Stop()
			return
		}
	}
}
