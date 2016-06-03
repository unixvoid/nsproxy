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

debug
-----
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
