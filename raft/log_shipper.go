package raft

import (
	"context"
	"net/rpc"
	"sync"

	log "github.com/sirupsen/logrus"
)

type LogShipper struct {
}

func (n *Node) AppendEntries(args AppendEntriesArgs, res *AppendEntriesResponse) error {
	log.Infof("%s got AppendEntry from %s", n.id, args.LeaderId)
	// Reset the election timer
	n.electionTimer.Reset(n.electionTimeout)

	n.mu.Lock()
	defer n.mu.Unlock()

	n.election = nil
	n.candidacy = nil
	n.state = Follower

	return nil
}

func (n *Node) AppendAll() error {
	wg := sync.WaitGroup{}

	for _, peer := range n.peers {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			if err := n.callAppendEntries(p); err != nil {
				log.Errorf("callAppendEntries failed for peer %q: %v", p, err)
			}
		}(peer)
	}

	wg.Wait()

	return nil
}

func (n *Node) callAppendEntries(addr string) error {
	log.Infof("%s calling AppendEntry to %s", n.id, addr)
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		log.Infof("RPC Client failed: %v\n", err)
		return err
	}

	args := AppendEntriesArgs{
		Term:     n.term,
		LeaderId: n.id,
	}

	res := &AppendEntriesResponse{}

	if err := client.Call("Raft.AppendEntries", args, res); err != nil {
		log.Infof("Failed to callRequestVote: %v", err)
		return err
	}

	return nil
}

func (n *Node) heartbeat(ctx context.Context) {
	for {
		select {
		case <-n.heartbeatTimer.C:
			log.Infof("%s doing heartbeat", n.id)
			if err := n.AppendAll(); err != nil {
				panic(err)
			}

			n.heartbeatTimer.Reset(n.heartbeatTimeout)
		case <-ctx.Done():
			log.Infof("%s heartbeat stopping", n.id)
			n.heartbeatTimer.Stop()
			return
		}
	}
}
