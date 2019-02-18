package raft

/*

TODO list:
* audit structure, add comments
* Finish append log logic to handle extraneous logs
* Make test cases for append log
* Finish other raft features (snapshot?)

* Refactor the indexes of the logs, particular the starting log has a value of -1 right now which is bad
* Do not do special cases for the very first log to append in the followers (see above with the -1 stuff)

*/
