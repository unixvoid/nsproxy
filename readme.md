Nsproxy
=======
[![Build Status (Travis)](https://travis-ci.org/unixvoid/nsproxy.svg?branch=develop)](https://travis-ci.org/unixvoid/nsproxy)  
Nsproxy is a DNS proxy and cluster manager written in go.  This project acts as
a normal DNS server (in addition to the cluster managment) and allows the use of
custom DNS entries.  Currently nsproxy fully supports A, AAAA, and CNAME
entries.

Documentation
=============
All documentation is in the [github wiki](https://github.com/unixvoid/nsproxy/wiki)
* [Configuration](https://github.com/unixvoid/nsproxy/wiki/Configuration)
* [API](https://github.com/unixvoid/nsproxy/wiki/API)
* [Basic Usage](https://github.com/unixvoid/nsproxy/wiki/Basic-Usage)
* [Building](https://github.com/unixvoid/nsproxy/wiki/Building)
* [Redis Usage](https://github.com/unixvoid/nsproxy/wiki/Redis-data-structures)

Quickstart
==========
To quickly get nsproxy up and running check out our page on [dockerhub](https://hub.docker.com/r/unixvoid/nsproxy/)
Or make sure you have [Golang](https://golang.org) and make installed, and use the following make commands:  
* `make deps` to pull down all the 'go gets'
* `make run` to run nsproxy!
