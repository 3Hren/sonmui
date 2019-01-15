#!/usr/bin/env make

# NOTE: variables defined with := in GNU make are expanded when they are
# defined rather than when they are used.
GOCMD := ./

# NOTE: variables defined with ?= sets the default value, which can be
# overriden using env.
GO ?= go
GOPATH ?= $(shell ls -d ~/go)

TARGETDIR := target
INSTALLDIR := ${GOPATH}/bin/

HOSTOS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOSTARCH := $(shell uname -m)

GOOS ?= ${HOSTOS}
GOARCH ?= ${HOSTARCH}

# Set the execution extension for Windows.
ifeq (${GOOS},windows)
    EXE := .exe
endif

OS_ARCH := $(GOOS)_$(GOARCH)$(EXE)

ICLI := ${TARGETDIR}/sonmicli_$(OS_ARCH)

TAGS = nocgo

.PHONY: fmt vet test

all: vet fmt build

build/icli:
	@echo "+ $@"
	${GO} build -tags "$(TAGS)" -ldflags "$(LDFLAGS)" -o ${ICLI} ${GOCMD}/icli

build: build/icli

install: all
	@echo "+ $@"
	mkdir -p ${INSTALLDIR}
	cp ${WORKER} ${CLI} ${NODE} ${INSTALLDIR}

vet:
	@echo "+ $@"
	@go tool vet $(shell ls -1 -d */ | grep -v -e vendor -e contracts)

fmt:
	@echo "+ $@"
	@test -z "$$(gofmt -s -l . 2>&1 | grep -v ^vendor/ | tee /dev/stderr)" || \
		(echo >&2 "+ please format Go code with 'gofmt -s'" && false)

clean:
	find . -name "*_mock.go" | xargs rm -f

deb:
	go mod download
	debuild --no-lintian --preserve-env -uc -us -i -I -b
	debuild clean
