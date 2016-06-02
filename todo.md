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


- add async server handler (will collect ips/register servers)
- add config file support

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

none	0
A		1
AAAA	28
CNAME	5
TXT		16
CAA		257
DHCID	49
