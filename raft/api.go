package raft

import (
	"encoding/json"
	"net/http"
)

func (n *Node) Status(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hey"))
}

func (n *Node) Write(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(); err != nil {

	}
}
