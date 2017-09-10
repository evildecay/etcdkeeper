## etcdkeeper
* Lightweight etcd web client.
* Support etcd 2.x and etcd 3.x.
* Server using the grpc interface, the server needs to compile the package etcd clientv3.
* Based easyui framework to achieve(easyui license [easyui website](http://www.jeasyui.com)).

## Usage
* Run httpserver.exe (windows version)
* If linux or darwin,Please compile it in its environment.
```
  Usage of httpserver.exe:  
  -h string  
        host name or ip address (default: "127.0.0.1", your machine addreess, not etcd address)
  -p int
        port (default 8080)
  -n string
        name (default: request, etcdv2 need to be used)
```
* Open your browser and enter the address. 
  - etcdv2: http://127.0.0.1:8080/etcdkeeper
  - etcdv3: http://127.0.0.1:8080/etcdkeeper3
* Right click on the tree node to add or delete.
* Etcd address can be modified by default to the localhost. If you change, press the Enter key to take effect.

## Features
* Etcd client view. Add, update or delete nodes.
* Content edits use the ace editor[(Ace editor)](https://ace.c9.io). Support toml,ini,yaml,json,xml and so on to highlight view.
* Content format. (Now only json, will later expand the new formatted content) Thanks jim3ma for his contribution. @jim3ma

## Special Note
Because the etcdv3 version uses the new storage concept, without the catalog concept, the client uses the previous default "/" delimiter to view. See the documentation for etcdv3 [clientv3 doc](https://godoc.org/github.com/coreos/etcd/clientv3).

## Screenshots
### etcdv2
![image](https://github.com/evildecay/etcdkeeper3/raw/master/screenshots/ui.png)
### etcdv3
![image](https://github.com/evildecay/etcdkeeper3/raw/master/screenshots/uiv3.png)

## License
MIT
