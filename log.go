package raft

import (
	"sync"

	"github.com/krantius/logging"
)

type Log struct {
	CurrentTerm  int
	CurrentIndex int
	CommitIndex  int
	logs         []LogEntry
	mu           sync.Mutex
}

type LogEntry struct {
	Term      int
	Index     int
	Committed bool
	Cmd       Command
}

// Append is called when a leader requests a follower to append entries
func (l *Log) Append(term, prevIndex, prevTerm, commitIndex int, entries []LogEntry) bool {
	if term < l.CurrentTerm {
		return false
	}

	if l.CurrentIndex < prevIndex {
		return false
	}

	if l.logs[prevIndex].Term != prevTerm {
		return false
	}

	difference := l.CurrentIndex - prevIndex
	if difference != 0 {
		// Trim off any old entries
		l.logs = l.logs[:len(l.logs)-difference]
	}

	logging.Debug("Accepted heartbeat")

	// Actually add the new entries
	l.append(entries)

	// Commit whatever is in the commitIndex
	l.commit(commitIndex)

	return true
}

func (l *Log) appendCmd(c Command) LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	le := LogEntry{
		Term:  l.CurrentTerm,
		Index: l.CurrentIndex,
		Cmd:   c,
	}

	l.logs = append(l.logs, le)

	l.CurrentIndex++

	return le
}

// append will just add to the entries and upate the term/index
func (l *Log) append(entries []LogEntry) {
	l.logs = append(l.logs, entries...)

	last := l.logs[len(l.logs)-1]
	l.CurrentIndex = last.Index
	l.CurrentTerm = last.Term
}

// commit will set all entries between the current index and the provided index
func (l *Log) commit(index int) {
	if l.CommitIndex >= index {
		return
	}

	for i := l.CommitIndex; i <= index; i++ {
		l.logs[i].Committed = true
	}

	l.CommitIndex = index
}
