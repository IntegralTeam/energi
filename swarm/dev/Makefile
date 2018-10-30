.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/IntegralTeam/energi/cmd/swarm
	go build -o bin/geth     github.com/IntegralTeam/energi/cmd/geth
	go build -o bin/bootnode github.com/IntegralTeam/energi/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/IntegralTeam/energi/swarm/...
