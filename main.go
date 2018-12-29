package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/krantius/definitely-not-raft/raft"
)

/*
type State string

const (
	Follower  State = "follower"
	Candidate State = "candidate"
	Leader    State = "leader"
)

type Server struct {
	// Persistent State
	currentTerm  int
	votedFor     string
	logs         []raft.LogEntry
	id           string
	listenerHost string

	// Volatile State on all servers
	commitIndex       uint64
	lastApplied       uint64
	lastAppendEntryMS int64
	votes             int
	state             State

	// Volatile State on leaders
	nextIndex  map[string]uint64
	matchIndex map[string]uint64

	// Connection Stuff
	s       *rpc.Server
	c       *rpc.Client
	cluster map[string]string

	// Concurrency
	mu              sync.Mutex
	electionTimeout time.Duration
	electionTimer   *time.Timer
}

func NewServer(id string, c *Config) *Server {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	to := time.Duration(r1.Intn(20)) * time.Second
	fmt.Println(to)
	s := &Server{
		id:              id,
		electionTimeout: to,
		state:           Follower,
		cluster:         make(map[string]string),
	}

	// Populate the addresses of our known buddies
	for _, server := range c.Servers {
		if server.ID != id {
			s.cluster[server.ID] = server.Addr
		} else {
			s.listenerHost = server.Addr
		}
	}

	s.electionTimer = time.NewTimer(s.electionTimeout)

	s.s = rpc.NewServer()

	oldMux := http.DefaultServeMux
	mux := http.NewServeMux()
	http.DefaultServeMux = mux

	s.s.RegisterName("Raft", s)
	s.s.HandleHTTP("/", "/debug")

	http.DefaultServeMux = oldMux

	// Need to handle both rpc and regular http requests

	return s
}

func UnixMillisecond() int64 {
	return time.Now().Unix() / int64(time.Millisecond)
}

func (s *Server) AppendEntries(args raft.AppendEntriesArgs, res *raft.AppendEntriesResponse) error {
	s.electionTimer.Reset(s.electionTimeout)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastAppendEntryMS = UnixMillisecond()

	fmt.Println("Server Append")
	return nil
}

func (s *Server) RequestVote(args raft.RequestVoteArgs, res *raft.RequestVoteResponse) error {
	if args.Term < s.currentTerm {
		formatLog(s.id, fmt.Sprintf("term less than %s, not voting", args.CadidateId))

		// Term less than current term, old leader?
		fmt.Println("RequestVote term less than current term, replying false")

		res.VoteGranted = false
		res.Term = s.currentTerm

		return nil
	}

	if s.currentTerm == args.Term && s.votedFor != "" {
		formatLog(s.id, fmt.Sprintf("already voted for %s, not voting for %s", s.votedFor, args.CadidateId))

		// Already voted for someone this term, skipping
		res.VoteGranted = false
		res.Term = s.currentTerm

		return nil
	}

	formatLog(s.id, fmt.Sprintf("voted for %s", args.CadidateId))

	s.votedFor = args.CadidateId
	s.currentTerm = args.Term
	s.state = Follower

	res.VoteGranted = true
	res.Term = args.Term

	return nil
}

func (s *Server) callAppendEntries(entries []raft.LogEntry, addr string) {
	s.electionTimer.Stop()
	var err error
	s.c, err = rpc.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("RPC Client failed: %v\n", err)
		return
	}

	args := raft.AppendEntriesArgs{
		Term:         s.currentTerm,
		LeaderId:     s.votedFor,
		PrevLogIndex: s.lastApplied,
		PrevLogTerm:  s.currentTerm - 1,
		Entries:      entries,
		LeaderCommit: s.commitIndex,
	}

	response := &raft.AppendEntriesResponse{}
	if err := s.c.Call("Raft.AppendEntries", args, response); err != nil {
		fmt.Fprintf(os.Stderr, "RPC Raft.AppendEntries failed: %v\n", err)
	}
}

func (s *Server) callRequestVote(addr, id string) {
	var err error
	s.c, err = rpc.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("RPC Client failed: %v\n", err)
		return
	}

	args := raft.RequestVoteArgs{
		Term:         s.currentTerm,
		CadidateId:   s.id,
		LastLogIndex: s.lastApplied,
		LastLogTerm:  s.currentTerm - 1,
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
			formatLog(s.id, "won leadership")

			// Set state to leader
			s.state = Leader

			// Call empty append entries to all servers
			for _, addr := range s.cluster {
				s.callAppendEntries(nil, addr)
			}

			s.electionTimer.Stop()
		}

		return
	}
}

func (s *Server) timer(ctx context.Context) {
	for {
		select {
		case <-s.electionTimer.C:
			// if time since last heartbeat or update > some amount
			// -> start new term and request votes

			s.mu.Lock()

			if s.state != Follower {
				s.mu.Unlock()
				continue
			}

			// Increment term
			s.currentTerm++

			// Set state
			s.state = Candidate

			// Vote for itself
			s.votedFor = s.id

			s.mu.Unlock()

			// Request votes from all other servers
			for id, addr := range s.cluster {
				s.callRequestVote(addr, id)
			}

			s.electionTimer.Reset(s.electionTimeout)

		case <-ctx.Done():
			fmt.Println("All done")
			return
		}
	}
}

func (s *Server) do() {
	go s.timer(context.Background())
	s.listen()
}

func (s *Server) listen() {
	l, e := net.Listen("tcp", s.listenerHost)
	if e != nil {
		log.Fatal("listen error:", e)
	}

	s.s.Accept(l)
}*/

type Config struct {
	Nodes []raft.NodeConfig `json:"nodes"`
}

func LoadConfig(path string) *Config {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	data := make([]byte, 1024)
	size, err := f.Read(data)
	if err != nil {
		panic(err)
	}

	data = data[:size]

	c := &Config{}
	if err := json.Unmarshal(data, c); err != nil {
		panic(err)
	}

	return c
}

func main() {
	id := os.Getenv("NODE_ID")
	if id == "" {
		panic("NODE_ID not set")
	}

	peers := os.Getenv("NODE_PEERS")

	port := 8001

	portArgs := os.Getenv("NODE_PORT")
	if portArgs != "" {
		tport, err := strconv.Atoi(portArgs)
		if err == nil {
			port = tport
		}
	}

	raftNode := raft.NewNode(id, port, strings.Split(peers, ","))
	go raftNode.Do()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	<-c
	/*
		c := LoadConfig("./config.json")
		fmt.Println(c)

		list := []*raft.Node{}
		for _, node := range c.Nodes {
			list = append(list, raft.NewNode(node.ID, c.Nodes))
		}

		wg := sync.WaitGroup{}
		for _, s := range list {
			wg.Add(1)
			go s.Do()
		}

		wg.Wait()
	*/

	/*id := os.Getenv("ID")

	s := NewServer(id, c)

	go s.electionTimer(context.Background())
	s.start()*/
}

func formatLog(id string, msg string) {
	fmt.Printf("id=%s %s\n", id, msg)
}
