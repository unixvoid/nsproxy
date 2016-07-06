GOC=go build
GOFLAGS=-a -ldflags '-s'
CGOR=CGO_ENABLED=0
IMAGE_NAME=nsproxy
DOCKER_DNS_LISTEN_PORT=53
DOCKER_API_LISTEN_PORT=8080
REDIS_DB_HOST_DIR=/tmp/
DOCKER_OPTIONS="--no-cache"
HOST_IP=192.168.1.9

all: nsproxy

nsproxy:
	$(GOC) nsproxy/*.go

run:
	go run nsproxy/*.go

rundocker:
	sudo docker run \
			-d \
			-p $(DOCKER_DNS_LISTEN_PORT):53/tcp \
			-p $(DOCKER_DNS_LISTEN_PORT):53/udp \
			-p $(DOCKER_API_LISTEN_PORT):8080/tcp \
			--name nsproxy \
			-v $(REDIS_DB_HOST_DIR):/redisbackup/:rw \
			$(IMAGE_NAME)
	sudo docker logs -f nsproxy

stage:
	make stat
	mkdir -p stage.tmp/
	mv bin/nsproxy stage.tmp/

stat:
	mkdir -p bin/
	$(CGOR) $(GOC) $(GOFLAGS) -o bin/nsproxy nsproxy/*.go

docker:
	$(MAKE) stat
	mkdir stage.tmp/
	cp bin/nsproxy stage.tmp/
	cp deps/rootfs.tar.gz stage.tmp/
	cp deps/Dockerfile stage.tmp/
	chmod +x deps/run.sh
	cp deps/run.sh stage.tmp/
	cp config.gcfg stage.tmp/
	cd stage.tmp/ && \
		sudo docker build $(DOCKER_OPTIONS) -t $(IMAGE_NAME) .
	@echo "$(IMAGE_NAME) built"

install: stat
	cp nsproxy /usr/bin/

link:
	mkdir -p $(GOPATH)/src/github.com/unixvoid/
	ln -s $(shell pwd) $(GOPATH)/src/github.com/unixvoid/

deps:
	go get github.com/gorilla/mux
	go get gopkg.in/gcfg.v1
	go get github.com/unixvoid/glogger
	go get github.com/unixvoid/nsproxy/nsmanager
	go get github.com/miekg/dns
	go get gopkg.in/redis.v3
	go get github.com/tatsushid/go-fastping

populate:
	curl -d dnstype=A -d domain=unixvoid.com. -d value=1.2.3.4 localhost:8080/dns
	curl -d dnstype=CNAME -d domain=unixvoid.com. -d value=turbo.lb.unixvoid.com localhost:8080/dns
	curl -d dnstype=AAAA -d domain=unixvoid.com. -d value=a111::a222:a333:a444:a555 localhost:8080/dns

testhealthcheck:
	./deps/runpod.sh $(HOST_IP)

testhealthcheckoneshot:
	./deps/runpodoneshot.sh $(HOST_IP)

rmhealthcheck:
	sudo docker stop testapp0
	sudo docker stop testapp1
	sudo docker stop testapp2
	sudo docker rm testapp0
	sudo docker rm testapp1
	sudo docker rm testapp2

rmhealthcheckoneshot:
	sudo docker stop testapp0
	sudo docker rm testapp0

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
	rm -rf bin/
	rm -f builddeps/nsproxy
	rm -rf stage.tmp/

#CGO_ENABLED=0 go build -a -ldflags '-s' nsproxy.go
