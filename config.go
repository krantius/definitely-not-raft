package raft

type Config struct {
	ID    string
	Port  int
	Peers []string
}
