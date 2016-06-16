GOC=go build
GOFLAGS=-a -ldflags '-s'
CGOR=CGO_ENABLED=0
IMAGE_NAME=nsproxy

all: nsproxy

nsproxy: nsproxy.go
	$(GOC) nsproxy.go

run:
	make stat
	sudo ./nsproxy

stage: nsproxy.go
	make stat
	make statremote
	mv nsproxy builddeps/

stat: nsproxy.go
	$(CGOR) $(GOC) $(GOFLAGS) nsproxy.go

docker:
	$(MAKE) stat
	mkdir stage.tmp/
	cp nsproxy stage.tmp/
	cp deps/rootfs.tar.gz stage.tmp/
	cp deps/Dockerfile stage.tmp/
	cp deps/run.sh stage.tmp/
	cp config.gcfg stage.tmp/
	cd stage.tmp/ && \
		sudo docker build -t $(IMAGE_NAME) .

install: stat
	cp nsproxy /usr/bin/

link:
	mkdir -p $(GOPATH)/src/git.unixvoid.com/mfaltys/
	ln -s $(shell pwd) $(GOPATH)/src/git.unixvoid.com/mfaltys/

deps:
	go get github.com/gorilla/mux
	go get gopkg.in/gcfg.v1
	go get git.unixvoid.com/mfaltys/glogger
	go get git.unixvoid.com/mfaltys/nsproxy/nsmanager
	go get github.com/miekg/dns
	go get gopkg.in/redis.v3
	go get github.com/tatsushid/go-fastping

test:
	@echo "----------------------------------------------------------------------"
	dig +noall +question +answer @localhost -p 8053 unixvoid.com.
	@echo "----------------------------------------------------------------------"
	dig +noall +question +answer @localhost -p 8053 unixvoid.com. A
	@echo "----------------------------------------------------------------------"
	dig +noall +question +answer @localhost -p 8053 unixvoid.com. AAAA
	@echo "----------------------------------------------------------------------"
	dig +noall +question +answer @localhost -p 8053 unixvoid.com. CNAME
	@echo "----------------------------------------------------------------------"
	@echo "testing complete"

clean:
	rm -f nsproxy
	rm -f builddeps/nsproxy
	rm -rf stage.tmp/

#CGO_ENABLED=0 go build -a -ldflags '-s' nsproxy.go
