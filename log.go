package raft

import (
	"sync"

	"github.com/krantius/logging"
)

// Log is the raft log used to store log entries
type Log struct {
	CurrentTerm     int
	CurrentIndex    int
	CommitIndex     int
	RequiredCommits int
	fsm             Store
	logs            []LogEntry
	mu              sync.RWMutex
}

// LogEntry is a single operation that gets applied to the FSM
type LogEntry struct {
	Term    int
	Index   int
	Commits int
	Cmd     Command
}

// Append is called when a leader requests a follower to append entries
func (l *Log) Append(term, prevIndex, prevTerm, commitIndex int, entries []LogEntry) bool {
	logging.Infof("term=%d prevIndex=%d prevTerm=%d commitIndex=%d entriesLen=%d", term, prevIndex, prevTerm, commitIndex, len(entries))
	if term < l.CurrentTerm {
		return false
	}

	l.CurrentTerm = term

	// First one
	if entries[0].Index == 0 && l.CurrentIndex == -1 {
		l.append(entries)
		return true
	}

	if l.CurrentIndex < prevIndex {
		logging.Errorf("Current Index %d < prevIndex %d", l.CurrentIndex, prevIndex)
		return false
	}

	if l.logs[prevIndex].Term != prevTerm {
		logging.Errorf("PrevIndex %d term %d not equal %d", prevIndex, l.logs[prevIndex].Term, prevTerm)
		return false
	}

	difference := l.CurrentIndex - prevIndex
	if difference != 0 {
		// If the current index is greater than the previous index from the leader, we have extraneous entries
		// Trim them
		l.logs = l.logs[:len(l.logs)-difference]
	}

	// Actually add the new entries
	l.append(entries)

	// Commit whatever is in the commitIndex
	// TODO currently only committing on heartbeats
	//l.commit(commitIndex)

	logging.Tracef("Applied term=%d index=%d commit=%d", term, l.CurrentIndex, l.CommitIndex)

	return true
}

func (l *Log) appendCmd(c Command) LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.CurrentIndex++

	le := LogEntry{
		Term:  l.CurrentTerm,
		Index: l.CurrentIndex,
		Cmd:   c,
	}

	l.logs = append(l.logs, le)

	return le
}

// append will just add to the entries and upate the term/index
func (l *Log) append(entries []LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logs = append(l.logs, entries...)

	last := l.logs[len(l.logs)-1]
	l.CurrentIndex = last.Index
	l.CurrentTerm = last.Term

	logging.Tracef("Appended the goods, %+v", l.logs)
}

// commit will set all entries between the current index and the provided index
func (l *Log) commit(index int) []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.CommitIndex >= index {
		return nil
	}

	committedLogs := []LogEntry{}
	for i := l.CommitIndex + 1; i <= index; i++ {
		committedLogs = append(committedLogs, l.logs[i])
	}

	l.CommitIndex = index
	logging.Infof("Committed index=%d %+v", index, committedLogs)

	return committedLogs
}

// TODO go on another thread
// Or do this one level higher
func (l *Log) commitCb(index int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	prev := l.CommitIndex
	for i := l.CommitIndex + 1; i <= index; i++ {
		l.logs[i].Commits++

		if l.logs[i].Commits >= l.RequiredCommits {
			// Relying on the fact that all log entries will be committed sequentially
			l.CommitIndex = i
		}
	}

	if prev != l.CommitIndex {
		for i := prev + 1; i <= l.CommitIndex; i++ {
			cmd := l.logs[i].Cmd
			switch cmd.Op {
			case Set:
				l.fsm.Set(cmd.Key, cmd.Val)
			case Delete:
				l.fsm.Delete(cmd.Key)
			}
		}
	}
}

func (l *Log) walk(le LogEntry) LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if le.Index == 0 {
		// Probably won't get here, but might as well check
		logging.Warning("Tried to walk on the very first log entry")
		return LogEntry{
			Index: -1,
		}
	}

	return l.logs[le.Index-1]
}

func (l *Log) get(index int) LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.logs[index]
}

// Range returns a range of LogEntry objects given a start and end period
func (l *Log) Range(start, end int) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if start == -1 {
		return l.logs[:end]
	}

	if end == -1 {
		return l.logs[start:]
	}

	return l.logs[start:end]
}
