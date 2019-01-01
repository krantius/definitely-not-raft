package raft

// Store is implemented by the client representing the underlying data store
type Store interface {
	Set(key string, val []byte)
	Delete(key string)
}

type Operation string

const (
	Set    Operation = "set"
	Delete Operation = "delete"
)

type Command struct {
	Op  Operation
	Key string
	Val []byte
}
