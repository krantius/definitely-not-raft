package raft

import (
	"context"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/krantius/logging"
)

type peerState string

const (
	stateSynced  peerState = "synced"
	stateSyncing peerState = "syncing"
)

type peer struct {
	addr    string
	current LogEntry
	l       *Log
	mu      sync.Mutex
	state   peerState
	ctx     context.Context
}

func (p *peer) heartbeat(term, commit int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state == stateSyncing {
		return nil
	}

	conn, err := net.DialTimeout("tcp", p.addr, appendTimeout)
	if err != nil {
		logging.Infof("RPC Client dial failed: %v\n", err)
		return err
	}

	client := rpc.NewClient(conn)

	res := AppendEntriesResponse{}
	args := AppendEntriesArgs{
		Term:         term,
		LeaderCommit: commit,
		PrevLogIndex: p.current.Index,
		PrevLogTerm:  p.current.Term,
		Entries:      nil,
	}

	if err := client.Call("Raft.AppendEntries", args, &res); err != nil {
		logging.Infof("Failed to callRequestVote: %v", err)
		return err
	}

	if !res.Success {
		// Resync...
		p.state = stateSyncing
		p.current = p.l.walk(p.current)
		go p.sync(term, commit)
		return nil
	}

	p.state = stateSynced

	return nil
}

func (p *peer) appendLog(term, commit int, entries []LogEntry) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state == stateSyncing {
		return nil
	}

	conn, err := net.DialTimeout("tcp", p.addr, appendTimeout)
	if err != nil {
		logging.Infof("RPC Client dial failed: %v\n", err)
		return err
	}

	client := rpc.NewClient(conn)

	res := AppendEntriesResponse{}
	args := AppendEntriesArgs{
		Term:         term,
		LeaderCommit: commit,
		PrevLogIndex: p.current.Index,
		PrevLogTerm:  p.current.Term,
		Entries:      entries,
	}

	if err := client.Call("Raft.AppendEntries", args, &res); err != nil {
		logging.Infof("Failed to callRequestVote: %v", err)
		return err
	}

	if !res.Success {
		// Resync...
		p.state = stateSyncing
		p.current = p.l.walk(p.current)
		go p.sync(term, commit)
		return nil
	}

	p.current = entries[len(entries)-1]
	p.state = stateSynced

	// Call commit CB or something
	p.l.commitCb(p.current.Index)

	return nil
}

// TODO make this a mega function that listens on a channel for append entries, has timer for heartbeats, and ctx done?
func (p *peer) sync(term, commit int) error {
	timer := time.NewTimer(0)
	timeout := time.Second

	for {
		select {
		case <-timer.C:
			conn, err := net.DialTimeout("tcp", p.addr, appendTimeout)
			if err != nil {
				timer.Reset(timeout)
				logging.Infof("RPC Client dial failed: %v\n", err)

				break
			}

			client := rpc.NewClient(conn)

			entries := p.l.Range(p.current.Index+1, -1)
			res := AppendEntriesResponse{}
			args := AppendEntriesArgs{
				Term:         term,
				LeaderCommit: commit,
				PrevLogIndex: p.current.Index,
				PrevLogTerm:  p.current.Term,
				Entries:      entries,
			}

			logging.Infof("Attempting to catch up with prevInde=%d prevTerm=%d", p.current.Index, p.current.Term)

			if err := client.Call("Raft.AppendEntries", args, &res); err != nil {
				timer.Reset(timeout)
				logging.Infof("Failed to callRequestVote: %v", err)

				break
			}

			if !res.Success {
				p.current = p.l.walk(p.current)
				timer.Reset(0)
				break
			}

			p.current = entries[len(entries)-1]

			p.mu.Lock()
			p.state = stateSynced
			p.mu.Unlock()

			p.l.commitCb(p.current.Index)
			return nil
		case <-p.ctx.Done():
			logging.Info("Stopping catchUp due to ctx cancel")
			return p.ctx.Err()
		}
	}
}
