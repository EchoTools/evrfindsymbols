
BIN=evrfindsymbols
BINEXE=$(BIN).exe
OBJS=$(BIN) $(BIN).exe

VERSION:=$(shell git describe --tags --always --dirty=+)
COMMIT:=$(shell git rev-parse --short HEAD)
PWD=$(shell pwd)

export version=${VERSION}
export commit=${COMMIT}

GCFLAGS=-gcflags "all=-trimpath ${PWD}" -asmflags "all=-trimpath ${PWD}"
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION} -X main.commitID=${COMMIT}"
ASMFLAGS=-asmflags "all=-trimpath ${PWD}"

DEPS=vendor/modules.txt main.go

.PHONY: all clean vendor


all: $(OBJS)

vendor/modules.txt: go.mod
	GOWORK=off go mod vendor

# Makefile for evrcat

$(BIN): $(DEPS)
	GOWORK=off GOOS=linux go build -o $(BIN) -trimpath -mod=vendor ${GCFLAGS} ${LDFLAGS} main.go

$(BINEXE): $(DEPS)
	GOWORK=off GOOS=windows go build -o $(BINEXE) -trimpath -mod=vendor ${GCFLAGS} ${LDFLAGS} main.go

clean:
	rm -f $(BIN)

