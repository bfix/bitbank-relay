
VERSION = $(shell git describe --tags --abbrev=0)
ifeq ($(VERSION),)
    VERSION = 0.0.0
endif

all: bitbank-relay-configurator bitbank-relay-db bitbank-relay-web

lib := $(wildcard lib/*.go)

bitbank-relay-configurator: configurator/main.go configurator/config-template.json $(lib)
	go build -o $@ -ldflags "-X main.Version=$(VERSION)" relay/configurator
	strip --strip-all $@

bitbank-relay-db: db/main.go db/gui.go db/logo.go db/gui.htpl $(lib)
	go build -o $@ -ldflags "-X main.Version=$(VERSION)" relay/db
	strip --strip-all $@

bitbank-relay-web: web/main.go web/service.go web/periodic.go $(lib)
	go build -o $@ -ldflags "-X main.Version=$(VERSION)" relay/web
	strip --strip-all $@