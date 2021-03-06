NAME := genji

.PHONY: all $(NAME) test testrace build gen

all: $(NAME)

build: $(NAME)

$(NAME):
	cd ./cmd/$@ && go install

gen: $(NAME)
	go generate ./...

test:
	go test -cover -timeout=1m ./...
	cd cmd/genji && go test -cover -timeout=1m ./...
	cd engine/badgerengine/ && go test -cover -timeout=1m ./...

testrace:
	go test -race -cover -timeout=1m ./...
	cd cmd/genji && go test -race -cover -timeout=1m ./...
	cd engine/badgerengine/ && go test -race -cover -timeout=1m ./...

testtinygo:
	go test -tags=tinygo -cover -timeout=1m ./...

bench:
	go test -v -run=^\$$ -benchmem -bench=. ./...
	cd cmd/genji && go test -v -run=^\$$ -benchmem -bench=. ./...

tidy:
	go mod tidy
	cd engine/badgerengine && go mod tidy && cd ../..
	cd cmd/genji && go mod tidy && cd ../..
