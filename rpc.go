package raft

// AppendEntriesArgs contains a request to add log entries to a raft node
type AppendEntriesArgs struct {
	Term         int
	LeaderID     string
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

// AppendEntriesResponse is a follower's response to an a leader's AppendEntry request
type AppendEntriesResponse struct {
	Term    int
	Success bool
}

// RequestVoteArgs is a candidate's request to other raft nodes during an election
type RequestVoteArgs struct {
	Term         int
	CadidateID   string
	LastLogIndex uint64
	LastLogTerm  int
}

// RequestVoteResponse is a follower's response to a candidate's vote request
type RequestVoteResponse struct {
	Term        int
	VoteGranted bool
}
