
VERSION = $(shell git describe --tags --abbrev=0)
ifeq ($(VERSION),)
    VERSION = 0.0.0
endif

bitbank-relay-configurator: configurator/main.go configurator/config-template.json
	go build -o bitbank-relay-configurator -ldflags "-X main.Version=$(VERSION)" relay/configurator

bitbank-relay-db: db/main.go db/gui.go db/logo.go db/gui.htpl
	go build -o bitbank-relay-db -ldflags "-X main.Version=$(VERSION)" relay/db
