Nsproxy
=======
[![Build Status (Travis)](https://travis-ci.org/unixvoid/nsproxy.svg?branch=develop)](https://travis-ci.org/unixvoid/nsproxy)  
Nsproxy is a DNS proxy and cluster manager written in go.  This project acts as
a normal DNS server (in addition to the cluster managment) and allows the use of
custom DNS entries.  Currently nsproxy fully supports A, AAAA, and CNAME
entries.

Documentation
=============
All documentation is in the [github wiki](https://unixvoid.github.io/nsproxy)
* [Configuration](https://unixvoid.github.io/nsproxy/configuration/)
* [API](https://unixvoid.github.io/nsproxy/api/)
* [Basic Usage](https://unixvoid.github.io/nsproxy/basic_usage/)
* [Building](https://unixvoid.github.io/nsproxy/building/)
* [Redis Usage](https://unixvoid.github.io/nsproxy/redis_data_structures/)

Quickstart
==========
To quickly get nsproxy up and running check out our page on [dockerhub](https://hub.docker.com/r/unixvoid/nsproxy/)
Or make sure you have [Golang](https://golang.org) and make installed, and use the following make commands:  
* `make deps` to pull down all the 'go gets'
* `make run` to run nsproxy!
