GOC=go build
GOFLAGS=-a -ldflags '-s'
CGOR=CGO_ENABLED=0

all: nsproxy

nsproxy: nsproxy.go
	$(GOC) nsproxy.go

run: nsproxy.go
	go run nsproxy.go

stage: nsproxy.go
	make stat
	make statremote
	mv nsproxy builddeps/
	mv remotemanager builddeps/

stat: nsproxy.go
	$(CGOR) $(GOC) $(GOFLAGS) nsproxy.go

statremote: remotemanager.go
	$(CGOR) $(GOC) $(GOFLAGS) remotemanager.go

install: stat
	cp nsproxy /usr/bin

link:
	mkdir -p $(GOPATH)/src/git.unixvoid.com/mfaltys/
	ln -s $(shell pwd) $(GOPATH)/src/git.unixvoid.com/mfaltys/

clean:
	rm -f nsproxy
	rm -f remotemanager
	rm -f localmanager
	rm -f builddeps/nsproxy
	rm -f builddeps/remotemanager
	rm -f builddeps/localmanager

#CGO_ENABLED=0 go build -a -ldflags '-s' nsproxy.go
