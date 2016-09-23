GOC=go build
GOFLAGS=-a -ldflags '-s'
CGOR=CGO_ENABLED=0
IMAGE_NAME=docker.io/unixvoid/nsproxy
HOST_LISTEN_PORT=8053
DOCKER_DNS_LISTEN_PORT=53
DOCKER_API_LISTEN_PORT=8080
REDIS_DB_HOST_DIR=/tmp/
DOCKER_OPTIONS="--no-cache"
HOST_IP=192.168.1.9
GIT_HASH=$(shell git rev-parse HEAD | head -c 10)

all: nsproxy

nsproxy:
	$(GOC) nsproxy/*.go

run:
	cd nsproxy && go run \
		api_clusters.go \
		api_cluster_spec.go \
		api_dns_entries.go \
		api_dns.go \
		api_dns_rm.go \
		api_dns_spec.go \
		api_hosts.go \
		api_host_spec.go \
		diff.go \
		dns_builder.go  \
		endpoints.go \
		new_host.go \
		nsproxy.go \
		sync.go \
		watch.go

daemon:
	bin/nsproxy &

test:
	go test -v nsproxy/*.go

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
	$(CGOR) $(GOC) $(GOFLAGS) -o bin/nsproxy-$(GIT_HASH)-linux-amd64 nsproxy/*.go

docker:
	$(MAKE) stat
	mkdir stage.tmp/
	cp bin/nsproxy* stage.tmp/
	cp deps/rootfs.tar.gz stage.tmp/
	cp deps/Dockerfile stage.tmp/
	sed -i "s/<DIFF>/$(GIT_HASH)/g" stage.tmp/Dockerfile
	chmod +x deps/run.sh
	cp deps/run.sh stage.tmp/
	cp nsproxy/config.gcfg stage.tmp/
	cd stage.tmp/ && \
		sudo docker build $(DOCKER_OPTIONS) -t $(IMAGE_NAME) .
	@echo "$(IMAGE_NAME) built"

aci:
	$(MAKE) stat
	mkdir -p stage.tmp/nsproxy-layout/rootfs/
	tar -zxf deps/rootfs.tar.gz -C stage.tmp/nsproxy-layout/rootfs/
	cp bin/nsproxy* stage.tmp/nsproxy-layout/rootfs/nsproxy
	chmod +x deps/run.sh
	cp deps/run.sh stage.tmp/nsproxy-layout/rootfs/
	sed -i "s/\$DIFF/$(GIT_HASH)/g" stage.tmp/nsproxy-layout/rootfs/run.sh
	cp nsproxy/config.gcfg stage.tmp/nsproxy-layout/rootfs/
	cp deps/manifest.json stage.tmp/nsproxy-layout/manifest
	cd stage.tmp/ && \
		actool build nsproxy-layout nsproxy.aci && \
		mv nsproxy.aci ../
	@echo "nsproxy.aci built"

testaci:
	deps/testrkt.sh

install: stat
	cp nsproxy /usr/bin/

link:
	mkdir -p $(GOPATH)/src/github.com/unixvoid/
	ln -s $(shell pwd) $(GOPATH)/src/github.com/unixvoid/

dependencies:
	go get github.com/gorilla/mux
	go get gopkg.in/gcfg.v1
	go get github.com/unixvoid/glogger
	go get github.com/unixvoid/nsproxy/pkg/nsmanager
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

testdns:
	dig +noall +answer @localhost -p $(HOST_LISTEN_PORT) cluster-testapp
	dig +noall +answer @localhost -p $(HOST_LISTEN_PORT) cluster-testapp
	dig +noall +answer @localhost -p $(HOST_LISTEN_PORT) cluster-testapp
	dig +noall +answer @localhost -p $(HOST_LISTEN_PORT) cluster-testapp
	dig +noall +answer @localhost -p $(HOST_LISTEN_PORT) cluster-testapp
	dig +noall +answer @localhost -p $(HOST_LISTEN_PORT) cluster-testapp
	dig +noall +answer @localhost -p $(HOST_LISTEN_PORT) cluster-testapp
	dig +noall +answer @localhost -p $(HOST_LISTEN_PORT) cluster-testapp
	dig +noall +answer @localhost -p $(HOST_LISTEN_PORT) cluster-testapp

testdig:
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

travisaci:
	wget https://github.com/appc/spec/releases/download/v0.8.7/appc-v0.8.7.tar.gz
	tar -zxf appc-v0.8.7.tar.gz
	$(MAKE) stat
	mkdir -p stage.tmp/nsproxy-layout/rootfs/
	tar -zxf deps/rootfs.tar.gz -C stage.tmp/nsproxy-layout/rootfs/
	cp bin/nsproxy* stage.tmp/nsproxy-layout/rootfs/nsproxy
	chmod +x deps/run.sh
	cp deps/run.sh stage.tmp/nsproxy-layout/rootfs/
	sed -i "s/\$DIFF/$(GIT_HASH)/g" stage.tmp/nsproxy-layout/rootfs/run.sh
	cp nsproxy/config.gcfg stage.tmp/nsproxy-layout/rootfs/
	cp deps/manifest.json stage.tmp/nsproxy-layout/manifest
	cd stage.tmp/ && \
		../appc-v0.8.7/actool build nsproxy-layout nsproxy.aci && \
		mv nsproxy.aci ../
	@echo "nsproxy.aci built"

clean:
	rm -rf bin/
	rm -f builddeps/nsproxy
	rm -f nsproxy.aci
	rm -rf stage.tmp/

#CGO_ENABLED=0 go build -a -ldflags '-s' nsproxy.go
