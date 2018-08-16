package main

type Raft interface {
	AppendEntries(term int, leaderId string, prevLogIndex uint64, prevLogTerm int, entries []LogEntry, leaderCommit string) (int, bool)
	RequestVote(term int, candidateId string, lastLogIndex, lastLogTerm uint64) (int, bool)
}


