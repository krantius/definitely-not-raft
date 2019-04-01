package raft

import (
	"reflect"
	"testing"
)

type mockFSM struct {
}

func (f mockFSM) Set(key string, val []byte) {}

func (f mockFSM) Delete(key string) {}

func TestAppend(t *testing.T) {
	cases := []struct {
		name           string
		testLog        *Log
		args           AppendEntriesArgs
		expectedResult bool
		expectedLog    *Log
	}{
		{
			name: "First Append",
			testLog: &Log{
				CurrentTerm:  0,
				CommitIndex:  -1,
				CurrentIndex: -1,
				fsm:          mockFSM{},
			},
			args: AppendEntriesArgs{
				Term:         1,
				PrevLogIndex: -1,
				PrevLogTerm:  0,
				Entries: []LogEntry{
					{
						Term:    1,
						Index:   0,
						Commits: 0,
						Cmd: Command{
							Key: "a",
							Val: []byte("abc"),
							Op:  Set,
						},
					},
				},
			},
			expectedResult: true,
			expectedLog: &Log{
				CommitIndex:  -1,
				CurrentTerm:  1,
				CurrentIndex: 0,
				logs: []LogEntry{
					{
						Term:    1,
						Index:   0,
						Commits: 0,
						Cmd: Command{
							Key: "a",
							Val: []byte("abc"),
							Op:  Set,
						},
					},
				},
				fsm: mockFSM{},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := c.testLog.Append(c.args.Term, c.args.PrevLogIndex, c.args.PrevLogTerm, c.args.LeaderCommit, c.args.Entries)

			if res != c.expectedResult {
				t.Errorf("Expected append to return false, but found true")
			}

			if c.testLog.CurrentTerm != c.expectedLog.CurrentTerm {
				t.Errorf("Incorrect CurrentTerm, expected=%d got=%d", c.expectedLog.CurrentTerm, c.testLog.CurrentTerm)
			}

			if c.testLog.CurrentIndex != c.expectedLog.CurrentIndex {
				t.Errorf("Incorrect CurrentIndex, expected=%d got=%d", c.expectedLog.CurrentIndex, c.testLog.CurrentIndex)
			}

			if c.testLog.CommitIndex != c.expectedLog.CommitIndex {
				t.Errorf("Incorrect CommitIndex, expected=%d got=%d", c.expectedLog.CommitIndex, c.testLog.CommitIndex)
			}

			if !reflect.DeepEqual(c.testLog.logs, c.expectedLog.logs) {
				t.Errorf("Log entries incorrect.\nExpected = %+v\nGot=%+v", c.expectedLog.logs, c.testLog.logs)
			}
		})
	}
}
