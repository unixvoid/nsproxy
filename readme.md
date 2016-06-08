nsproxy
=======

nsproxy is a DNS proxy and cluster manager written in go.  This project acts as
a normal DNS server (in addition to the cluster managment) and allows the use of
custom DNS entries.  Currently nsproxy fully supports A, AAAA, and CNAME
entries.

### configuration  
- nsproxy uses gcfg (INI-style config files for go structs).  The config uses some pretty sane defauls but the following fields are configurable:  
- `[server]`
  - `port:`  the port the main DNS server listens on.  
  - `loglevel:`  the verbosity of logs. acceptable fields are 'info', 'cluster', 'debug', and 'error'.  
- `[clustermanager]`
  - `useclustermanager:`  whether or not to use the cluster manager. acceptable fields are 'true' and 'false'  
  - `port:`  the port that cluster manager will listen on (this is what port clients use to check in)  
  - `pingfeq:`  the ammout of time in between health checks (in seconds)  
- `[dns]`
  - `ttl:`  the default time to live (in seconds) for dns entries  
- `[upstreamdns]`
  - `server:`  the dns server and port that nsproxy uses if it cannot find a match in the local database  
- `[redis]`  
  - `host:`  this is the ip and port that the redis backend is running on  
  - `password:`  password to the redis database if one exists

### nsproxy usage
- The following usage implies the default config file is being used.  
- On boot nsproxy will bind to two ports:  
  - `8053` is used as the regular dns server.  This will act the same as any other dns server and allows for custom dns entries to be used.  
  - `8080` is used as the cluster manager.  Clients should post to this port to bind with the dns server.  
- DNS entries can be added in a similar fashion to registering a host.  A POST on the same port that the clustermanager is running on in the following format will add an entry to the dns server.  
  - `/dns` dnstype= domain= value=  
    - example `curl -d dnstype=a domain=unixvoid.com value=192.168.1.80 localhost:8080/dns`  
  - `/dns/rm` dnstype= domain= will remove a dns entry (and all for a domain if 'dnstype' not set)  
    - example `curl -d dnstype=a domain=unixvoid.com localhost:8080/dns/rm`  
  - `dns:<dns_type>:<fqdn>` and the content being a valid A, AAAA, or CNAME entry.  
- Here are some examples on what typical redis entries would look like.  
  - entry: `dns:a:unixvoid.com.` content: `67.3.192.22`  
  - entry: `dns:aaaa:unixvoid.com.` content: `::1`  
  - entry: `dns:cname:unixvoid.com.` content: `customlb.cname.`  
- To register a client with the cluster manager, the client will send a form
    (`application/x-www-form-urlencoded`) to nsproxy with the following data.
    - `hostname`:  the hostname of the box  
    - `cluster`:  the intended cluster to join.  
    - `ip`(optional):  ip (usefull when client is behind proxy/loadbalancer).  
  - Both of these fields are required.  
- A regular client registration looks like this:  
    `curl -d hostname=nginx -d cluster=coreos unixvoid.com:8080`  This will add the host `nginx` to the cluster `coreos`.  These names are arbitrary and can be anything.  

### api
- nsproxy exposes an api for easy access to data and creating DNS records.  The following is the specification for endpoints and their protocols.  
- `/` : `POST` : endpoint for registering hosts to the dns loadbalancer.  
  - `hostname` : hostname of the box
  - `cluster` : cluster the host is associated with
  - `ip` : (optional) the ip the box is on, you only need to set this if the client is behind a proxy/loadbalancer
  - example: `curl -d hostname=$1 -d cluster=smartos 192.168.2.201:8080`
- `/dns` : `POST`: endpoint for adding a DNS entry to the db
  - `dnstype` : the type of request being made: supported {a, aaaa, cname}
  - `domain` : domain name, this will be fully qualified by nsproxy if it is not already
  - `value` : the entry value {ip4 address for 'a', ipv6 address for 'aaaa', aname for 'aname'}
  - example: `curl -d dnstype=CNAME -d domain=bitnuke.io -d value=turbo.lb.bitnuke.io localhost:8080/dns`
- `/dns/rm` : `POST` : endpoint to remove a DNS entry from the db
  - `dnstype` : (optional) dns request to remove {a, aaaa, cname}. If not set, will remove all three from the db for the domain
  - `domain` : the domain to remove entry for
  - example: `curl -d dnstype=a -d domain=bitnuke.io localhost:8080/dns/rm`
- `/custerspec` : `POST` : get hosts for a specific cluster
  - `cluster` : cluster name to get hosts for
  - example: `curl -d cluster=smartos localhost:8080/clusterspec`
- `/hostspec` : `POST` : get the ip of a specific host
  - `cluster` : cluster name that the host belongs to
  - `host` : hostname to get the ip of
  - example: `curl -d cluster=smartos -d host=test1 localhost:8080/hostspec`
- `/hosts` : `GET` : get a list of all live hosts (in the format <cluster>:<host>)
  - example: `curl localhost:8080/hosts`
- `/clusters` : `GET` : get a list of all live clusters
  - example: `curl localhost:8080/clusters`  

### building
- This project requires golang to be installed with the dependencies in place.
- To pull the dependencies on your box simply issue `make deps` to do all the `go get`s for you.  
- make will accept the following commands:  
  - `make nsproxy` will dynamically build nsproxy
  - `make stat` will statically build nsproxy
  - `make stage` will build and stage all files for preperation of building a
     container
  - `make install` will install the compiled nsproxy in /usr/bin/
  - `make clean` will clean the project of all tmp directories and binaries

### redis
- This is designed to help the user understand what is going on inside of redis at a low level.  The following are redis keys (with examples) and what they store.
- `cluster:<cluster_name>:<host_name>` This is a low level entry that has a hostname's ip
  - type: redis key
    - content: host ip
    - example `cluster:coreos:nginx` 192.168.2.2
- `index:cluster:<cluster_name>` This is a medium level entry that contains the elements (hosts) that are in a cluster
  - type: redis set
  - content: cluster hosts
  - example `index:cluster:coreos` {nginx, cApp, configServer}
- `list:cluster:<cluster_name>` This is a clone of the previous, used to hold order or load balanced hosts. This element stays in order to obey load balancer algoritms
  - type: redis unordered set (list)
  - content: cluster hosts
  - example `list:cluster:coreos` {nginx, cApp, configServer}
- `index:master` This a persistent index that hold a list of all clusters (this persists across nsproxy reboots)
  - type: redis set
  - content: clusters
  - example `index:master` {coreos, neatCluster, ps2_cluster}
- `index:live` This is volitile entry that gets diffed against `index:master` and only contains hosts that have live listeners
  - type: redis set
  - content: clusters
  - example `index:master` {coreos, neatCluster, ps2_cluster}
- `dns:<dns_type>:<domain>` This is the standard dns entry  
  - type: redis set  
  - content: a, aaaa, cname entry  
  - example `dns:a:unixvoid.com.` 192.168.1.80  
