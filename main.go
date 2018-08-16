package main

import (
	"net/rpc"
	"net"
	"log"
)

type LogEntry struct {
	index uint64
	term int
	data []byte
}

type Server struct {
	// Persistent State
	currentTerm int
	votedFor string
	logs []LogEntry

	// Volatile State on all servers
	commitIndex uint64
	lastApplied uint64

	// Volatile State on leaders
	nextIndex map[string]uint64
	matchIndex map[string]uint64
}

func (s *Server) AppendEntries(term int, leaderId string, prevLogIndex uint64, prevLogTerm int, entries []LogEntry, leaderCommit string) (int, bool) {
	return 1, true
}

func (s *Server) RequestVote(term int, candidateId string, lastLogIndex, lastLogTerm uint64) (int, bool) {
	return 1, true
}

func main() {
	server := rpc.NewServer()
	server.RegisterName("Raft", Server{})

	server.HandleHTTP("/", "/debug")

	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}

	server.Accept(l)
}
