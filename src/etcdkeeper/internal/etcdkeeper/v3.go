package etcdkeeper

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"
	"google.golang.org/grpc"
)

// v3 api
func (ek *Etcdkeeper) Connect(w http.ResponseWriter, r *http.Request) {
	ek.mu.Lock()
	defer ek.mu.Unlock()
	sess := ek.sessmgr.SessionStart(w, r)
	host := r.FormValue("host")
	uname := r.FormValue("uname")
	passwd := r.FormValue("passwd")

	if ek.config.useAuth {
		if _, ok := ek.rootUsers[host]; !ok && uname != "root" { // no root user
			b, _ := json.Marshal(map[string]interface{}{"status": "root"})
			io.WriteString(w, string(b))
			return
		}
		if uname == "" || passwd == "" {
			b, _ := json.Marshal(map[string]interface{}{"status": "login"})
			io.WriteString(w, string(b))
			return
		}
	}

	if uinfo, ok := sess.Get("uinfo").(*userInfo); ok {
		if host == uinfo.host && uname == uinfo.uname && passwd == uinfo.passwd {
			info := ek.getInfo(host)
			b, _ := json.Marshal(map[string]interface{}{"status": "running", "info": info})
			io.WriteString(w, string(b))
			return
		}
	}

	uinfo := &userInfo{host: host, uname: uname, passwd: passwd}
	c, err := ek.newClient(uinfo)
	if err != nil {
		log.Println(r.Method, "v3", "connect fail.")
		b, _ := json.Marshal(map[string]interface{}{"status": "error", "message": err.Error()})
		io.WriteString(w, string(b))
		return
	}
	defer c.Close()
	_ = sess.Set("uinfo", uinfo)

	if ek.config.useAuth {
		if uname == "root" {
			ek.rootUsers[host] = uinfo
		}
	} else {
		ek.rootUsers[host] = uinfo
	}
	log.Println(r.Method, "v3", "connect success.")
	info := ek.getInfo(host)
	b, _ := json.Marshal(map[string]interface{}{"status": "running", "info": info})
	io.WriteString(w, string(b))
}

func (ek *Etcdkeeper) Put(w http.ResponseWriter, r *http.Request) {
	cli := ek.getClient(w, r)
	defer cli.Close()
	key := r.FormValue("key")
	value := r.FormValue("value")
	ttl := r.FormValue("ttl")
	log.Println("PUT", "v3", key)

	var err error
	data := make(map[string]interface{})
	if ttl != "" {
		var sec int64
		sec, err = strconv.ParseInt(ttl, 10, 64)
		if err != nil {
			log.Println(err.Error())
		}
		var leaseResp *clientv3.LeaseGrantResponse
		leaseResp, err = cli.Grant(context.TODO(), sec)
		if err == nil && leaseResp != nil {
			_, err = cli.Put(context.Background(), key, value, clientv3.WithLease(leaseResp.ID))
		}
	} else {
		_, err = cli.Put(context.Background(), key, value)
	}
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if resp, err := cli.Get(context.Background(), key); err != nil {
			data["errorCode"] = 500
			data["errorCode"] = err.Error()
		} else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = ek.getTTL(cli, kv.Lease)
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				data["node"] = node
			}
		}
	}

	var dataByte []byte
	if dataByte, err = json.Marshal(data); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func (ek *Etcdkeeper) Get(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})
	key := r.FormValue("key")
	log.Println("GET", "v3", key)

	var cli *clientv3.Client
	sess := ek.sessmgr.SessionStart(w, r)
	v := sess.Get("uinfo")
	var uinfo *userInfo

	if v != nil {
		uinfo = v.(*userInfo)
		cli, _ = ek.newClient(uinfo)
		defer cli.Close()

		permissions, e := ek.getPermissionPrefix(uinfo.host, uinfo.uname, key)
		if e != nil {
			io.WriteString(w, e.Error())
			return
		}
		if r.FormValue("prefix") == "true" {
			pnode := make(map[string]interface{})
			pnode["key"] = key
			pnode["nodes"] = make([]map[string]interface{}, 0)
			for _, p := range permissions {
				var (
					resp *clientv3.GetResponse
					err  error
				)
				if p[1] != "" {
					prefixKey := p[0]
					if p[0] == "/" {
						prefixKey = ""
					}
					resp, err = cli.Get(context.Background(), prefixKey, clientv3.WithPrefix())
				} else {
					resp, err = cli.Get(context.Background(), p[0])
				}
				if err != nil {
					data["errorCode"] = 500
					data["message"] = err.Error()
				} else {
					for _, kv := range resp.Kvs {
						node := make(map[string]interface{})
						node["key"] = string(kv.Key)
						node["value"] = string(kv.Value)
						node["dir"] = false
						if key == string(kv.Key) {
							node["ttl"] = ek.getTTL(cli, kv.Lease)
						} else {
							node["ttl"] = 0
						}
						node["createdIndex"] = kv.CreateRevision
						node["modifiedIndex"] = kv.ModRevision
						nodes := pnode["nodes"].([]map[string]interface{})
						pnode["nodes"] = append(nodes, node)
					}
				}
			}
			data["node"] = pnode
		} else {
			if resp, err := cli.Get(context.Background(), key); err != nil {
				data["errorCode"] = 500
				data["message"] = err.Error()
			} else {
				if resp.Count > 0 {
					kv := resp.Kvs[0]
					node := make(map[string]interface{})
					node["key"] = string(kv.Key)
					node["value"] = string(kv.Value)
					node["dir"] = false
					node["ttl"] = ek.getTTL(cli, kv.Lease)
					node["createdIndex"] = kv.CreateRevision
					node["modifiedIndex"] = kv.ModRevision
					data["node"] = node
				} else {
					data["errorCode"] = 500
					data["message"] = "The node does not exist."
				}
			}
		}
	}

	var dataByte []byte
	var err error
	if dataByte, err = json.Marshal(data); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func (ek *Etcdkeeper) GetPath(w http.ResponseWriter, r *http.Request) {
	originKey := r.FormValue("key")
	log.Println("GET", "v3", originKey)
	var (
		data = make(map[string]interface{})
		/*
			{1:["/"], 2:["/foo", "/foo2"], 3:["/foo/bar", "/foo2/bar"], 4:["/foo/bar/test"]}
		*/
		all = make(map[int][]map[string]interface{})
		min int
		max int
		//prefixKey string
	)

	var cli *clientv3.Client
	sess := ek.sessmgr.SessionStart(w, r)
	v := sess.Get("uinfo")
	var uinfo *userInfo
	if v != nil {
		uinfo = v.(*userInfo)
		cli, _ = ek.newClient(uinfo)
		defer cli.Close()

		permissions, e := ek.getPermissionPrefix(uinfo.host, uinfo.uname, originKey)
		if e != nil {
			io.WriteString(w, e.Error())
			return
		}

		// parent
		var (
			presp *clientv3.GetResponse
			err   error
		)
		if originKey != ek.config.separator {
			presp, err = cli.Get(context.Background(), originKey)
			if err != nil {
				data["errorCode"] = 500
				data["message"] = err.Error()
				dataByte, _ := json.Marshal(data)
				io.WriteString(w, string(dataByte))
				return
			}
		}
		if originKey == ek.config.separator {
			min = 1
			//prefixKey = separator
		} else {
			min = len(strings.Split(originKey, ek.config.separator))
			//prefixKey = originKey
		}
		max = min
		all[min] = []map[string]interface{}{{"key": originKey}}
		if presp != nil && presp.Count != 0 {
			all[min][0]["value"] = string(presp.Kvs[0].Value)
			all[min][0]["ttl"] = ek.getTTL(cli, presp.Kvs[0].Lease)
			all[min][0]["createdIndex"] = presp.Kvs[0].CreateRevision
			all[min][0]["modifiedIndex"] = presp.Kvs[0].ModRevision
		}
		all[min][0]["nodes"] = make([]map[string]interface{}, 0)

		for _, p := range permissions {
			key, rangeEnd := p[0], p[1]
			//child
			var resp *clientv3.GetResponse
			if rangeEnd != "" {
				resp, err = cli.Get(context.Background(), key, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
			} else {
				resp, err = cli.Get(context.Background(), key, clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
			}
			if err != nil {
				data["errorCode"] = 500
				data["message"] = err.Error()
				dataByte, _ := json.Marshal(data)
				io.WriteString(w, string(dataByte))
				return
			}

			for _, kv := range resp.Kvs {
				if string(kv.Key) == ek.config.separator {
					continue
				}
				keys := strings.Split(string(kv.Key), ek.config.separator) // /foo/bar
				for i := range keys {                                      // ["", "foo", "bar"]
					k := strings.Join(keys[0:i+1], ek.config.separator)
					if k == "" {
						continue
					}
					node := map[string]interface{}{"key": k}
					if node["key"].(string) == string(kv.Key) {
						node["value"] = string(kv.Value)
						if key == string(kv.Key) {
							node["ttl"] = ek.getTTL(cli, kv.Lease)
						} else {
							node["ttl"] = 0
						}
						node["createdIndex"] = kv.CreateRevision
						node["modifiedIndex"] = kv.ModRevision
					}
					level := len(strings.Split(k, ek.config.separator))
					if level > max {
						max = level
					}

					if _, ok := all[level]; !ok {
						all[level] = make([]map[string]interface{}, 0)
					}
					levelNodes := all[level]
					var isExist bool
					for _, n := range levelNodes {
						if n["key"].(string) == k {
							isExist = true
						}
					}
					if !isExist {
						node["nodes"] = make([]map[string]interface{}, 0)
						all[level] = append(all[level], node)
					}
				}
			}
		}

		// parent-child mapping
		for i := max; i > min; i-- {
			for _, a := range all[i] {
				for _, pa := range all[i-1] {
					if i == 2 {
						pa["nodes"] = append(pa["nodes"].([]map[string]interface{}), a)
						pa["dir"] = true
					} else {
						if strings.HasPrefix(a["key"].(string), pa["key"].(string)+ek.config.separator) {
							pa["nodes"] = append(pa["nodes"].([]map[string]interface{}), a)
							pa["dir"] = true
						}
					}
				}
			}
		}
	}
	data = all[min][0]
	if dataByte, err := json.Marshal(map[string]interface{}{"node": data}); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func (ek *Etcdkeeper) Del(w http.ResponseWriter, r *http.Request) {
	cli := ek.getClient(w, r)
	defer cli.Close()
	key := r.FormValue("key")
	dir := r.FormValue("dir")
	log.Println("DELETE", "v3", key)

	if _, err := cli.Delete(context.Background(), key); err != nil {
		io.WriteString(w, err.Error())
		return
	}

	if dir == "true" {
		if _, err := cli.Delete(context.Background(), key+ek.config.separator, clientv3.WithPrefix()); err != nil {
			io.WriteString(w, err.Error())
			return
		}
	}
	io.WriteString(w, "ok")
}

func (ek *Etcdkeeper) getTTL(cli *clientv3.Client, lease int64) int64 {
	resp, err := cli.Lease.TimeToLive(context.Background(), clientv3.LeaseID(lease))
	if err != nil {
		return 0
	}
	if resp.TTL == -1 {
		return 0
	}
	return resp.TTL
}

func (ek *Etcdkeeper) getClient(w http.ResponseWriter, r *http.Request) *clientv3.Client {
	sess := ek.sessmgr.SessionStart(w, r)
	v := sess.Get("uinfo")
	if v != nil {
		uinfo := v.(*userInfo)
		c, _ := ek.newClient(uinfo)
		return c
	}
	return nil
}

func (ek *Etcdkeeper) newClient(uinfo *userInfo) (*clientv3.Client, error) {
	endpoints := []string{uinfo.host}
	var err error

	// use tls if usetls is true
	var tlsConfig *tls.Config
	if ek.config.usetls {
		tlsInfo := transport.TLSInfo{
			CertFile:      ek.config.cert,
			KeyFile:       ek.config.keyfile,
			TrustedCAFile: ek.config.cacert,
		}
		tlsConfig, err = tlsInfo.ClientConfig()
		if err != nil {
			log.Println(err.Error())
		}
	}

	conf := clientv3.Config{
		Endpoints:   endpoints,
		TLS:         tlsConfig,
		DialTimeout: time.Second * time.Duration(ek.config.connectTimeout),
		DialOptions: []grpc.DialOption{grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ek.config.grpcMaxMsgSize))},
	}
	if ek.config.useAuth {
		conf.Username = uinfo.uname
		conf.Password = uinfo.passwd
	}

	var c *clientv3.Client
	c, err = clientv3.New(conf)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (ek *Etcdkeeper) getPermissionPrefix(host, uname, key string) ([][]string, error) {
	if !ek.config.useAuth {
		return [][]string{{key, "p"}}, nil // No auth return all
	} else {
		if uname == "root" {
			return [][]string{{key, "p"}}, nil
		}
		rootUser := ek.rootUsers[host]
		rootCli, err := ek.newClient(rootUser)
		if err != nil {
			return nil, err
		}
		defer rootCli.Close()

		if resp, err := rootCli.UserList(context.Background()); err != nil {
			return nil, err
		} else {
			// Find user permissions
			set := make(map[string]string)
			for _, u := range resp.Users {
				if u == uname {
					ur, err := rootCli.UserGet(context.Background(), u)
					if err != nil {
						return nil, err
					}
					for _, r := range ur.Roles {
						rr, err := rootCli.RoleGet(context.Background(), r)
						if err != nil {
							return nil, err
						}
						for _, p := range rr.Perm {
							set[string(p.Key)] = string(p.RangeEnd)
						}
					}
					break
				}
			}
			var pers [][]string
			for k, v := range set {
				pers = append(pers, []string{k, v})
			}
			return pers, nil
		}
	}
}

func (ek *Etcdkeeper) getInfo(host string) map[string]string {
	info := make(map[string]string)
	uinfo := ek.rootUsers[host]
	rootClient, err := ek.newClient(uinfo)
	if err != nil {
		log.Println(err)
		return info
	}
	defer rootClient.Close()

	status, err := rootClient.Status(context.Background(), host)
	if err != nil {
		log.Fatal(err)
	}
	mems, err := rootClient.MemberList(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	kb := 1024
	mb := kb * 1024
	gb := mb * 1024
	var sizeStr string
	for _, m := range mems.Members {
		if m.ID == status.Leader {
			info["version"] = status.Version
			gn, rem1 := ek.size(int(status.DbSize), gb)
			mn, rem2 := ek.size(rem1, mb)
			kn, bn := ek.size(rem2, kb)
			if sizeStr != "" {
				sizeStr += " "
			}
			if gn > 0 {
				info["size"] = fmt.Sprintf("%dG", gn)
			} else {
				if mn > 0 {
					info["size"] = fmt.Sprintf("%dM", mn)
				} else {
					if kn > 0 {
						info["size"] = fmt.Sprintf("%dK", kn)
					} else {
						info["size"] = fmt.Sprintf("%dByte", bn)
					}
				}
			}
			info["name"] = m.GetName()
			break
		}
	}
	return info
}
