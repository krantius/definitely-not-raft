package replication

type Log struct {
	CurrentTerm  int
	CurrentIndex int
	CommitIndex  int
	entries      []entry
}

type entry struct {
	Term      int
	Index     int
	Committed bool
	Data      []byte
}

func (l *Log) Append(term, prevIndex, prevTerm, commitIndex int, entries []entry) bool {
	if term < l.CurrentTerm {
		return false
	}

	if l.CurrentIndex < prevIndex {
		return false
	}

	if l.entries[prevIndex].Term != prevTerm {
		return false
	}

	difference := l.CurrentIndex - prevIndex
	if difference != 0 {
		// Trim off any old entries
		l.entries = l.entries[:len(l.entries)-difference]
	}

	// Actually add the new entries
	l.append(entries)

	// Commit whatever is in the commitIndex
	l.commit(commitIndex)

	return true
}

// append will just add to the entries and upate the term/index
func (l *Log) append(entries []entry) {
	l.entries = append(l.entries, entries...)

	last := l.entries[len(l.entries)-1]
	l.CurrentIndex = last.Index
	l.CurrentTerm = last.Term
}

// commit will set all entries between the current index and the provided index
func (l *Log) commit(index int) {
	for i := l.CommitIndex; i <= index; i++ {
		l.entries[i].Committed = true
	}

	l.CommitIndex = index
}
