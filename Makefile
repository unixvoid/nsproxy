GOC=go build
GOFLAGS=-a -ldflags '-s'
CGOR=CGO_ENABLED=0

all: goNSproxy

goNSproxy: goNSproxy.go
	$(GOC) goNSproxy.go

run: goNSproxy.go
	go run goNSproxy.go

stat: goNSproxy.go
	$(CGOR) $(GOC) $(GOFLAGS) goNSproxy.go

install: stat
	cp goNSproxy /usr/bin

clean:
	rm -f goNSproxy

#CGO_ENABLED=0 go build -a -ldflags '-s' goNSproxy.go
