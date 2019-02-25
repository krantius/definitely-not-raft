package raft

// Store is implemented by the client representing the underlying data store
type Store interface {
	Set(key string, val []byte)
	Delete(key string)
}

// Command contains the instruction and key/value to apply to the FSM
type Command struct {
	Op  Operation
	Key string
	Val []byte
}

// Operation defines the action to take within a command
type Operation string

const (
	// Set is when a key gets set
	Set Operation = "set"

	// Delete is when a key gets deleted
	Delete Operation = "delete"
)
