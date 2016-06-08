TODO
------
DONE- dns entry api
  - dnstype=
  - domain=
  - value=
- dns remove entry api `/dns/rm`
- api handler to view:
  DONE- hosts
  DONE- clusters
  - hosts in a cluster (clusterspec)
- add ip field when registering (in case host is behind proxy/nsproxy picksup the wrong ip)
- impliment redis handler that throws exception when no redis connection

type reference
--------------
these are the typecodes for various lookup types in miekg/dns  
https://en.wikipedia.org/wiki/List_of_DNS_record_types  
https://github.com/miekg/dns/blob/master/types.go#L27  

```
none	0
A	1
AAAA	28
CNAME	5
TXT	16
CAA	257
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
