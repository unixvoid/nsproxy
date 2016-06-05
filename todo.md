TODO
------
- layout:  
  the new layout will look like the following when done:  
  - central server will register IPs/domain names and store them
  in a redis backed db.
  - when a server/application starts up it will send its domain name/role
  to nsproxy which will register it in the system.  
  - now when a dns request comes through nsproxy will check a local redis
  cache to resolve the proper IP based on a specified algorithm
  - impliment redis handler that throws exception when no redis connection


- add async server handler (will collect ips/register servers)
- add config file support
- add cluster tool to check online hosts (in event of nsproxy going down)
  to determine if host still in index are alive

redis data model
----------------
- cluster based storage
  - cluster:<clustername>:<hostname>
  - cluster:unixvoid:nginx
    - content: ip (comma seperated)

- dns based records
  - dns:<record type>:<hostname>
  - dns:a:unixvoid.com.
    - content: ip (comma seperated)
  - dns:cname:unixvoid.com.
    - content: alias
  - dns:url:unixvoid.com.
    - content: redirect url

type reference
--------------
these are the typecodes for various lookup types in miekg/dns  
https://en.wikipedia.org/wiki/List_of_DNS_record_types  
https://github.com/miekg/dns/blob/master/types.go#L27  

```
none	0
A		1
AAAA	28
CNAME	5
TXT		16
CAA		257
DHCID	49
```

debug reference
---------------
```
glogger.Debug.Println("---------------------------------------------------------")
glogger.Debug.Printf("ID :: %v", req.MsgHdr.Id)
glogger.Debug.Printf("NS :: %v", req.Ns)
glogger.Debug.Printf("Header :: %v", req.MsgHdr)
glogger.Debug.Printf("Compress :: %v", req.Compress)
glogger.Debug.Printf("Question :: %v", req.Question[0].Qtype)
glogger.Debug.Printf("Answer :: %v", req.Answer)
glogger.Debug.Printf("Extra :: %v", req.Extra)
glogger.Debug.Println("---------------------------------------------------------")
glogger.Debug.Printf("Req :: %v", req)
```

flow
----
```
- when a host comes on line, it gets a cluster entry: 
  - redis key: cluster:<cluster_name>:<host_name>
  - content:<ip>
- this then adds(creates if it does not exist) to an inex entry: 
  - redis key: index:cluster:<cluster_name>
  - content: <hostname>
- this will also add itself to an index of all live hosts:
  - redis key: index:master
  - content: <cluster_name>:<host_name>
- after these entries are in the db, it spawns a listener in charge of the host
  this listener will check the master live file:
  - redis key: index:live
  - content: <cluster_name>:<host_name>
- when nsproxy first comes on line it check the index:master entry and checks for live hosts.
  if a live host is found, it is added back into the 'index:live' entry
  - note that all entries in 'index:live' have a listener to them. every other
    host that is added to the 'index:master' after nsproxy starts will kick off a
    task to check diff the 'index:master' and 'index:live' entries. any squbsequent
    delta between the two will get a new listener.
  - every host that is offline or becomes offline will remove itself from the following:
    index:live, index:master, index:<cluster_name>:<host_name>, and cluster:<cluster_name>:<host_name>
  - if nsproxy dies mid process it it the job (next boot) to check all entries in 'index:master'
    to see if live host still exist (are live) if not, it will clean out their respective entries in
    'index:<cluster_name>:<host_name>' and 'cluster:<cluster_name>:<host_name>'
```
