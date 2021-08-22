.PHONY: all

all:
	docker image build -t etcdkeeper:1.0.0-snapshot .

lint:
	cd src/etcdkeeper && golangci-lint run --new-from-rev=HEAD~
