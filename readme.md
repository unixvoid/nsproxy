nsproxy
=======

nsproxy is a DNS proxy written in go.
This project acts as a normal DNS server but will allow
you to change spoof entries on the fly. Packaged in this
project is a remote manager that can be chained with the
nsproxy to update and add entries during runtime.

- **building**  
  this project requires golang if you want to build from source  
  to build the project you can clone it down with
  `git clone https://github.com/unixvoid/nsproxy`  
  and simply issue `go run nsproxy`.  
  The project is also in dockerhub `https://hub.docker.com/r/mfaltys/nsproxy/`  
  The dockerfile can be found in `builddeps` and on the dockerhub  

- **nsproxy**  
  nsproxy will take these arguments if you wish to specify:  
  `-p` this is the port the nsproxy listens on, default is 53  
  `-debug` this will start nsproxy with debug logs  
  `-upstream` this is the upstream DNS, default is 8.8.8.8:53  
  `-chain` This will run the remote manager along with the nsproxy
  this starts the remote manager on port 8054.  

- **remotemanager**  
  remotemanager is a tool used to list, add, remove, and modify custom
  entries in the nsproxy. Runtime flags are as follows:  
  `-p` this sets the listening port, default is 8054  
  commands can be curled from the terminal, or any other way that
  supports GET requests. The following commands are ones that can be
  issued in a web browser.  
  ```
  !list   : lists all entris
  !add    : adds a record
  !rm     : removes a record
  !modify : modifies a record
  ```

  examples:  
  `localhost:8054/!list` will list all entries  
  `localhost:8054/!new github.com 8.8.8.8` github.com will now resolve to 8.8.8.8  
  `localhost:8054/!rm github.com` will remove the github.com entry from our storage  

- **localmanager**  
  this is a simple tool to modify records locally, and takes all the same commands
  as remotemanager but locally. This is to be run in the same directory as nsproxy.  

  ```
  -list   : lists all entris
  -add    : adds a record
  -rm     : removes a record
  -modify : modifies a record
  ```

  examples:  
  `go run localmanager -list` will list all entries  
  `go run localmanager -new github.com 8.8.8.8` github.com will now resolve to 8.8.8.8  
  `go run localmanager -rm github.com` will remove the github.com entry from our storage  

- **other building procedures**  
  If you would like to build your own dockerfile, you can stage the built images
  in the `builddeps` directory and build the dockerfile from there. Common usage
  is as follows:  

  ```
  $make stage
  ..statically compiled binaries are moved to builddeps/
  $cd builddeps/
  $docker build -t nsproxy .
  ..build docker image 'nsproxy'
  $docker run -d -p 53:53 -p 8054:8054 nsproxy
  ..run daemonized container with webproxy running on 8054
  ```
