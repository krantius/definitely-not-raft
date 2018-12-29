build:
	GOOS=linux GOARCH=amd64 go build -o not-raft
docker: build
	docker build -f ./Dockerfile.1 -t not-raft:latest .
