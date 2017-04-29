## etcdkeeper v3
* Lightweight etcd web client.
* Support etcd 3.x
* Server using the grpc interface, the server needs to compile the package etcd clientv3
* Based easyui framework to achieve.(easyui license [easyui website](www.jeasyui.com))

## Usage
* Run httpserver3.exe  
```
  Usage of httpserver3.exe:  
  -h string  
        host name or ip address (default "127.0.0.1", your machine addreess, not etcd address)
  -p int
        port (default 8080)
```
* Open your browser and enter the address. http://127.0.0.1:8080/etcdkeeper3
* Right click on the tree node to add or delete
* Etcd address can be modified by default to the localhost. If you change, press the Enter key to take effect

## Recently want to achieve the new features
* Content edits use the editor. [Ace editor](https://ace.c9.io)

## Special Note
Because the etcdv3 version uses the new storage concept, without the catalog concept, the client uses the previous default "/" delimiter to view. See the documentation for etcdv3 [clientv3 doc](https://godoc.org/github.com/coreos/etcd/clientv3)

## Screenshots
![image](https://github.com/evildecay/etcdkeeper3/raw/master/screenshots/ui.png)

## License
MIT
