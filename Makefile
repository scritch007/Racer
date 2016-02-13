SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

BINARY=Racer

VERSION=1.0.0
BUILD_TIME=`date +%FT%T%z`

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(SOURCES)
	go build -o ${BINARY} main.go

.PHONY: install
install:
	go install ${LDFLAGS} ./...

.PHONY: clean
clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

.PHONY: pack
pack:
	tar czf racer.tar.gz html Racer
