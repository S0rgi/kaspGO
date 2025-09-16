PATH_MAIN=./cmd/app/main.go
EXEC_FILE=bin/app
DOC_FILE=docs/DOCUMENTATION.md
PATH_COVER= coverage
all:start

update:
	go mod tidy

linter:
	golangci-lint run

build:clean
	go build -o ${EXEC_FILE} ${PATH_MAIN}

start: build
	./${EXEC_FILE}
format:
	goimports -w internal/. cmd/.
clean:
	rm -rf ${EXEC_FILE} coverage/*
coverage:
	mkdir -p ${PATH_COVER}
	go test -race -coverprofile=${PATH_COVER}/coverage.out ./...
	go tool cover -func=${PATH_COVER}/coverage.out
coverage-html: coverage
	go tool cover -html=${PATH_COVER}/coverage.out -o ${PATH_COVER}/coverage.html
	xdg-open ${PATH_COVER}/coverage.html

doc:
	gomarkdoc ./... > ${DOC_FILE}
run:
	WORKERS=4 QUEUE_SIZE=64 ./${EXEC_FILE}

.PHONY: all update linter build start format clean coverage run