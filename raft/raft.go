package raft


type Raft interface {
	AppendEntries(args AppendEntriesArgs, res *AppendEntriesResponse) error
	RequestVote(args RequestVoteArgs, res *RequestVoteResponse) error
}

type LogEntry struct {
	Index uint64
	Term int
	Data []byte
}

type AppendEntriesArgs struct {
	Term int
	LeaderId string
	PrevLogIndex uint64
	PrevLogTerm int
	Entries []LogEntry
	LeaderCommit uint64
}

type AppendEntriesResponse struct {
	A int
	B bool
}

type RequestVoteArgs struct {
	Term int
	CadidateId string
	LastLogIndex uint64
	LastLogTerm int
}

type RequestVoteResponse struct {
	Term int // IS THIS RIGHT??
	VoteGranted bool
}

