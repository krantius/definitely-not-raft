package main

import (
	"net/rpc"
	"fmt"
	"net"
	"log"
	"github.com/krantius/definitely-not-raft/raft"
	"time"
	"context"
	"os"
	"sync"
)

type State string

const (
	Follower State = "follower"
	Candidate State = "candidate"
	Leader State = "leader"
)

type Server struct {
	// Persistent State
	currentTerm int
	votedFor string
	logs []raft.LogEntry
	id string

	// Volatile State on all servers
	commitIndex       uint64
	lastApplied       uint64
	lastAppendEntryMS int64
	electionTimeout int64
	votes int
	state State

	// Volatile State on leaders
	nextIndex map[string]uint64
	matchIndex map[string]uint64

	// Connection Stuff
	s *rpc.Server
	c *rpc.Client
	cluster map[string]string

	// Concurrency
	mu sync.Mutex
}

func NewServer(id string, c *Config) *Server {
	s := &Server{
		electionTimeout: int64(time.Second),
		state: Follower,
	}

	// Populate the addresses of our known buddies
	for _, server := range c.Servers {
		if server.ID != id {
			s.cluster[server.ID] = server.Addr
		}
	}

	s.s = rpc.NewServer()
	s.s.RegisterName("Raft", s)
	s.s.HandleHTTP("/", "/debug")

	// Need to handle both rpc and regular http requests

	return s
}

func UnixMillisecond() int64 {
	return time.Now().Unix() / int64(time.Millisecond)
}

func (s *Server) AppendEntries(args raft.AppendEntriesArgs, res *raft.AppendEntriesResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastAppendEntryMS = UnixMillisecond()

	fmt.Println("Server Append")
	return nil
}

func (s *Server) RequestVote(args raft.RequestVoteArgs, res *raft.RequestVoteResponse) error {
	if args.Term < s.currentTerm {
		// Term less than current term, old leader?
		fmt.Println("RequestVote term less than current term, replying false")

		res.VoteGranted = false
		res.Term = s.currentTerm

		return nil
	}

	if s.currentTerm == args.Term && s.votedFor != "" {
		// Already voted for someone this term, skipping
		res.VoteGranted = false
		res.Term = s.currentTerm

		return nil
	}

	s.votedFor = args.CadidateId
	s.currentTerm = args.Term
	s.state = Follower

	res.VoteGranted = true
	res.Term = args.Term

	return nil
}

func (s *Server) callAppendEntries(entries []raft.LogEntry, addr string) {
	var err error
	s.c, err = rpc.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("RPC Client failed: %v\n", err)
		return
	}

	args := raft.AppendEntriesArgs{
		Term: s.currentTerm,
		LeaderId: s.votedFor,
		PrevLogIndex: s.lastApplied,
		PrevLogTerm: s.currentTerm-1,
		Entries: entries,
		LeaderCommit: s.commitIndex,
	}

	response := &raft.AppendEntriesResponse{}
	if err := s.c.Call("Raft.AppendEntries", args, response); err != nil {
		fmt.Fprintf(os.Stderr, "RPC Raft.AppendEntries failed: %v\n", err)
	}
}

func (s *Server) callRequestVote(addr string) {
	var err error
	s.c, err = rpc.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("RPC Client failed: %v\n", err)
		return
	}

	args := raft.RequestVoteArgs{
		Term: s.currentTerm,
		CadidateId: s.id,
		LastLogIndex: s.lastApplied,
		LastLogTerm: s.currentTerm-1,
	}

	res := &raft.RequestVoteResponse{}

	if err := s.c.Call("Raft.RequestVote", args, res); err != nil {
		fmt.Printf("Failed to callRequestVote: %v\n", err)
	}

	if res.VoteGranted {
		// do stuff with the vote
		s.votes++

		// Check if we have majority votes
		// Only running 3 nodes rn
		if s.votes >= 2 {
			// We are the leader now, do leader stuff

			// Set state to leader
			s.state = Leader

			// Call empty append entries to all servers
			for _, addr := range s.cluster {
				s.callAppendEntries(nil, addr)
			}
		}

		return
	}
}

func (s *Server) electionTimer(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// if time since last heartbeat or update > some amount
			// -> start new term and request votes

			s.mu.Lock()

			if s.state != Follower {
				s.mu.Unlock()
				continue
			}

			if UnixMillisecond() - s.lastAppendEntryMS > s.electionTimeout {
				// Increment term
				s.currentTerm++

				// Set state
				s.state = Candidate

				// Vote for itself
				s.votedFor = s.id

				s.mu.Unlock()

				// Request votes from all other servers
				for _, addr := range s.cluster {
					s.callRequestVote(addr)
				}
			}

		case <-ctx.Done():
			fmt.Println("All done")
			return
		}
	}
}

func (s *Server) start() {
	port := os.Getenv("SERVER_PORT")

	l, e := net.Listen("tcp", "localhost:" + port)
	if e != nil {
		log.Fatal("listen error:", e)
	}

	s.s.Accept(l)
}

func main() {
	c := LoadConfig("./config.json")
	fmt.Println(c)
	/*id := os.Getenv("ID")

	s := NewServer(id, c)

	go s.electionTimer(context.Background())
	s.start()*/
}

