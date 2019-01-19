package raft

/*

TODO list:

* Handle locking better (better usage of mutexes)
* Retry committing old logs if a log does not get committed
* Refactor how the leader appends entries to the followers
	* How do we handle retries and resyncs? Is it an issue if a node is being resynced and it gets a heartbeat?
* Refactor the indexes of the logs, particular the starting log has a value of -1 right now which is bad
* Do not do special cases for the very first log to append in the followers (see above with the -1 stuff)

*/
