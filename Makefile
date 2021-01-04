PACKAGE = statsagent
VERSION_BASE ?= 0.0.1
VERSION_SUFFIX ?=
VERSION = ${VERSION_BASE}${VERSION_SUFFIX}
BUILD_NUMBER ?= 0
PACKAGE_DIR = ${PACKAGE}-${VERSION}

.PHONY: clean install check all

all: statsagent

statsagent: 
	go build -v -o statsagent

install: statsagent
	go install

clean:
	rm -f statsagent



