package raft

import (
	"context"
	"sync"
	"time"

	"github.com/krantius/logging"
)

const (
	appendTimeout time.Duration = 200 * time.Millisecond
)

// appendEntries is called when a follower gets a request from a leader
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

		// This means a node is behind and needs to be resynced
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

// appendAll sends out log entries to all the peers
// Leader fanning out to peers
func (r *Raft) appendAll(entries []LogEntry) {
	//r.mu.Lock()
	//defer r.mu.Unlock()

	logging.Tracef("Appending all %v", entries)

	wg := sync.WaitGroup{}

	for _, p := range r.peerLogs {
		wg.Add(1)

		go func(p *peer) {
			defer wg.Done()
			if err := p.AppendLog(r.term, r.log.CommitIndex, entries); err != nil {
				logging.Warningf("AppendLog failed for peer %s", p.addr)
			}
		}(p)
	}

	wg.Wait()
}

func (r *Raft) heartbeat(ctx context.Context) {
	for {
		select {
		case <-r.heartbeatTimer.C:
			r.election = nil

			wg := sync.WaitGroup{}

			r.mu.Lock()

			for _, p := range r.peerLogs {
				wg.Add(1)
				go func(p *peer) {
					defer wg.Done()
					if err := p.Heartbeat(r.term, r.log.CommitIndex); err != nil {
						logging.Warningf("Heartbeat failed for %s", p.addr)
					}
				}(p)
			}

			wg.Wait()
			r.mu.Unlock()

			r.heartbeatTimer.Reset(r.heartbeatTimeout)
		case <-ctx.Done():
			logging.Info("Heartbeat stopping")
			r.heartbeatTimer.Stop()
			return
		}
	}
}
