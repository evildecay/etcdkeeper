package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"etcdkeeper/session"
	_ "etcdkeeper/session/providers/memory"
	"flag"
	"fmt"
	"github.com/coreos/etcd/client"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	sep       = flag.String("sep", "/", "separator")
	separator = ""
	usetls    = flag.Bool("usetls", false, "use tls")
	cacert    = flag.String("cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle")
	cert      = flag.String("cert", "", "identify secure client using this TLS certificate file")
	keyfile   = flag.String("key", "", "identify secure client using this TLS key file")
	useAuth   = flag.Bool("auth", false, "use auth")
	clients   = make(map[string]*clientv3.Client)
	v2clients     = make(map[string]client.Client)

	sessmgr   *session.Manager
	mu        sync.Mutex
)

const (
	rootClientName = "_root_"
)

func main() {
	host := flag.String("h","0.0.0.0","host name or ip address")
	port := flag.Int("p", 8080, "port")

	flag.CommandLine.Parse(os.Args[1:])
	separator = *sep

	middleware := func(fns ...func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			for _, fn := range fns {
				fn(w, r)
			}
		}
	}

	// v2
	//http.HandleFunc(*name, v2request)
	http.HandleFunc("/v2/separator", middleware(nothing, getSeparator))
	http.HandleFunc("/v2/connect", middleware(nothing, connectV2))
	http.HandleFunc("/v2/put", middleware(nothing, putV2))
	http.HandleFunc("/v2/get", middleware(nothing, getV2))
	http.HandleFunc("/v2/delete", middleware(nothing, delV2))
	// dirctory mode
	http.HandleFunc("/v2/getpath", middleware(nothing, getPathV2))

	// v3
	http.HandleFunc("/v3/separator", middleware(nothing, getSeparator))
	http.HandleFunc("/v3/connect", middleware(nothing, connect))
	http.HandleFunc("/v3/put", middleware(nothing, put))
	http.HandleFunc("/v3/get", middleware(nothing, get))
	http.HandleFunc("/v3/delete", middleware(nothing, del))
	// dirctory mode
	http.HandleFunc("/v3/getpath", middleware(nothing, getPath))

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Session management
	sessmgr, err = session.NewManager("memory", "_etcdkeeper_session", 86400)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		ticker := time.NewTicker(86400*time.Second)
		for range ticker.C {
			sessmgr.GC()
			log.Println("Session GC.")
		}
	}()
	//log.Println(http.Dir(wd + "/assets"))

	http.Handle("/", http.FileServer(http.Dir(wd + "/assets"))) // view static directory

	log.Printf("listening on %s:%d\n", *host, *port)
	err = http.ListenAndServe(*host + ":" + strconv.Itoa(*port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func nothing(_ http.ResponseWriter, _ *http.Request) {
	// Nothing
}


//func v2request(w http.ResponseWriter, r *http.Request){
//	if err := r.ParseForm(); err != nil {
//		log.Println(err.Error())
//	}
//	log.Println(r.Method, "v2", r.FormValue("url"), r.PostForm.Encode())
//
//	body := strings.NewReader(r.PostForm.Encode())
//	req, err := http.NewRequest(r.Method, r.Form.Get("url"), body)
//	if err != nil {
//		io.WriteString(w, err.Error())
//		return
//	}
//	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
//	client := &http.Client{Timeout: 10*time.Second} // important!!!
//	resp, err := client.Do(req)
//	if err != nil {
//		io.WriteString(w, err.Error())
//	}else {
//		result, err := ioutil.ReadAll(resp.Body)
//		if err != nil {
//			io.WriteString(w, "Get data failed: " + err.Error())
//		} else {
//			io.WriteString(w, string(result))
//		}
//	}
//}

// v2 api
func connectV2(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	sess := sessmgr.SessionStart(w, r)
	host := strings.TrimSpace(r.FormValue("host"))
	uname := r.FormValue("uname")
	passwd := r.FormValue("passwd")
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}
	endpoints := []string{host}
	var info map[string]string
	if *useAuth {
		_, ok := v2clients[host + "@" + rootClientName]
		if !ok && uname != "root" {
			b, _ := json.Marshal(map[string]interface{}{"status":"root"})
			io.WriteString(w, string(b))
			return
		}
		if uname == "" || passwd == "" {
			b, _ := json.Marshal(map[string]interface{}{"status":"login"})
			io.WriteString(w, string(b))
			return
		}
	}

	if v := sess.Get("clientv2"); v != nil {
		c := v.(client.Client)
		if uname == sess.Get("username").(string) {
			if host == c.Endpoints()[0] {
				info = getInfoV2(host)
				b, _ := json.Marshal(map[string]interface{}{"status":"running", "info":info})
				io.WriteString(w, string(b))
				return
			}
		}
	} else {
		// v2 is not closed
	}

	cfg := client.Config{
		Endpoints:               endpoints,
		HeaderTimeoutPerRequest: 5*time.Second,
	}
	if *useAuth {
		cfg.Username = uname
		cfg.Password = passwd
	}

	c, err := client.New(cfg)
	if err != nil {
		log.Println(r.Method, "v2", "connect fail.")
		b, _ := json.Marshal(map[string]interface{}{"status":"error", "message":err.Error()})
		io.WriteString(w, string(b))
	} else {
		v2clients[host + "@" + uname] = c
		if *useAuth {
			if uname == "root" { // etcd must be root to manage
				v2clients[host + "@" + rootClientName] = c
			}
		} else {
			v2clients[host + "@" + rootClientName] = c
		}
		sess.Set("clientv2", c)
		sess.Set("clientName", host + "@" + uname)
		sess.Set("username", uname) // for v2
		log.Println(r.Method, "v2", "connect success.")
		info = getInfoV2(host)
		b, _ := json.Marshal(map[string]interface{}{"status":"ok", "info":info})
		io.WriteString(w, string(b))
	}
}

func putV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	value := r.FormValue("value")
	ttl := r.FormValue("ttl")
	dir := r.FormValue("dir")
	log.Println("PUT", "v2", key)

	kapi := client.NewKeysAPI(getClientV2(w, r))

	var isDir bool
	if dir != "" {
		isDir, _ = strconv.ParseBool(dir)
	}
	var err error
	data := make(map[string]interface{})
	if ttl != "" {
		var sec int64
		sec, err = strconv.ParseInt(ttl, 10, 64)
		if err != nil {
			log.Println(err.Error())
		}
		_, err = kapi.Set(context.Background(), key, value, &client.SetOptions{TTL:time.Duration(sec)*time.Second, Dir:isDir})
	} else {
		_, err = kapi.Set(context.Background(), key, value, &client.SetOptions{Dir:isDir})
	}
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if resp, err := kapi.Get(context.Background(), key, &client.GetOptions{Recursive:true, Sort:true}); err != nil {
			data["errorCode"] = err.Error()
		} else {
			if resp.Node != nil {
				node := make(map[string]interface{})
				node["key"] = resp.Node.Key
				node["value"] = resp.Node.Value
				node["dir"] = resp.Node.Dir
				node["ttl"] = resp.Node.TTL
				node["createdIndex"] = resp.Node.CreatedIndex
				node["modifiedIndex"] = resp.Node.ModifiedIndex
				data["node"] = node
			}
		}
	}

	var dataByte []byte
	if dataByte, err = json.Marshal(data);err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func getV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	data := make(map[string]interface{})
	log.Println("GET", "v2", key)

	kapi := client.NewKeysAPI(getClientV2(w, r))

	var permissions [][]string
	if r.FormValue("prefix") == "true" {
		var e error
		permissions, e = getPermissionPrefixV2(getClientName(w, r), key)
		if e != nil {
			io.WriteString(w, e.Error())
			return
		}
	} else {
		permissions = [][]string{{key, ""}}
	}

	var (
		min, max int
	)
	if key == separator {
		min = 1
	} else {
		min = len(strings.Split(key, separator))
	}
	max = min
	all := make(map[int][]map[string]interface{})
	if key == separator {
		all[min] = []map[string]interface{}{{"key":key, "value":"", "dir":true, "nodes":make([]map[string]interface{}, 0)}}
	}
	for _, p := range permissions {
		pKey, pRange := p[0], p[1]
		var opt *client.GetOptions
		if pRange != "" {
			if pRange == "c" {
				pKey += separator
			}
			opt = &client.GetOptions{Recursive:true, Sort:true}
		}
		if resp, err := kapi.Get(context.Background(), pKey, opt); err != nil {
			data["errorCode"] = 500
			data["message"] = err.Error()
		} else {
			if resp.Node == nil {
				data["errorCode"] = 500
				data["message"] = "The node does not exist."
			} else {
				max = getNode(resp.Node , key, all, min, max)
			}
		}
	}

	//b, _ := json.MarshalIndent(all, "", "  ")
	//fmt.Println(string(b))

	// parent-child mapping
	for i := max; i > min; i-- {
		for _, a := range all[i] {
			for _, pa := range all[i-1] {
				if i == 2 { // The last is root
					pa["nodes"] = append(pa["nodes"].([]map[string]interface{}), a)
					pa["dir"] = true
				} else {
					if strings.HasPrefix(a["key"].(string), pa["key"].(string) + separator) {
						pa["nodes"] = append(pa["nodes"].([]map[string]interface{}), a)
						pa["dir"] = true
					}
				}
			}
		}
	}

	for _, n := range all[min] {
		if n["key"] == key {
			nodesSort(n)
			data["node"] = n
			break
		}
	}

	var dataByte []byte
	var err error
	if dataByte, err = json.Marshal(data);err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func nodesSort(node map[string]interface{}) {
	if v, ok := node["nodes"]; ok && v != nil {
		a := v.([]map[string]interface{})
		if len(a) != 0 {
			for i := 0; i < len(a) - 1; i++ {
				nodesSort(a[i])
				for j := i + 1; j < len(a); j++ {
					if a[j]["key"].(string) < a[i]["key"].(string) {
						a[i], a[j] = a[j], a[i]
					}
				}
			}
			nodesSort(a[len(a) - 1])
		}
	}
}

func getNode(node *client.Node, selKey string, all map[int][]map[string]interface{}, min, max int) int {
	keys := strings.Split(node.Key, separator) // /foo/bar
	if len(keys) < min && strings.HasPrefix(node.Key, selKey) {
		return max
	}
	for i := range keys { // ["", "foo", "bar"]
		k := strings.Join(keys[0:i+1], separator)
		if k == "" {
			continue
		}
		nodeMap := map[string]interface{}{"key": k, "dir":true, "nodes":make([]map[string]interface{}, 0)}
		if k == node.Key {
			nodeMap["value"] = node.Value
			nodeMap["dir"] = node.Dir
			nodeMap["ttl"] = node.TTL
			nodeMap["createdIndex"] = node.CreatedIndex
			nodeMap["modifiedIndex"] = node.ModifiedIndex
		}
		keylevel := len(strings.Split(k, separator))
		if keylevel > max {
			max = keylevel
		}

		if _, ok := all[keylevel];!ok {
			all[keylevel] = make([]map[string]interface{}, 0)
		}
		var isExist bool
		for _, n := range all[keylevel] {
			if n["key"].(string) == k {
				isExist = true
			}
		}
		if !isExist {
			all[keylevel] = append(all[keylevel], nodeMap)
		}
	}

	if len(node.Nodes) != 0 {
		for _, n := range node.Nodes {
			max = getNode(n, selKey, all, min, max)
		}
	}
	return max
}

func delV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	dir := r.FormValue("dir")
	log.Println("DELETE", "v2", key)

	kapi := client.NewKeysAPI(getClientV2(w, r))

	isDir, _ := strconv.ParseBool(dir)
	if isDir {
		if _, err := kapi.Delete(context.Background(), key, &client.DeleteOptions{Recursive:true, Dir:true}); err != nil {
			io.WriteString(w, err.Error())
			return
		}
	} else {
		if _, err := kapi.Delete(context.Background(), key, nil); err != nil {
			io.WriteString(w, err.Error())
			return
		}
	}

	io.WriteString(w, "ok")
}

func getPathV2(w http.ResponseWriter, r *http.Request) {
	getV2(w, r)
}

func getClientV2(w http.ResponseWriter, r *http.Request) client.Client {
	sess := sessmgr.SessionStart(w, r)
	v := sess.Get("clientv2")
	if v != nil {
		return v.(client.Client)
	}
	return nil
}

func getPermissionPrefixV2(clientName, key string) ([][]string, error) {
	if !*useAuth {
		return [][]string{{key, "p"}}, nil // No auth return all
	} else {
		a := strings.Split(clientName, "@")
		host, uname := a[0], a[1]

		if uname == "root" {
			return [][]string{{key, "p"}}, nil
		}

		if !strings.HasPrefix(host, "http://") {
			host = "http://" + host
		}
		rootCli := v2clients[host + "@" + rootClientName]
		rootUserKapi := client.NewAuthUserAPI(rootCli)
		rootRoleKapi := client.NewAuthRoleAPI(rootCli)

		if users, err := rootUserKapi.ListUsers(context.Background()); err != nil {
			return nil, err
		} else {
			// Find user permissions
			set := make(map[string]string)
			for _, u := range users {
				if u == uname {
					user, err := rootUserKapi.GetUser(context.Background(), u)
					if err != nil {
						return nil, err
					}
					for _, r := range user.Roles {
						role, err := rootRoleKapi.GetRole(context.Background(), r)
						if err != nil {
							return nil, err
						}
						for _, ks := range role.Permissions.KV.Read {
							var k string
							if strings.HasSuffix(ks, "*") {
								k = ks[:len(ks) - 1]
								set[k] = "p"
							} else if strings.HasSuffix(ks, "/*") {
								k = ks[:len(ks) - 2]
								set[k] = "c"
							} else {
								if _, ok := set[ks]; !ok {
									set[ks] = ""
								}
							}
						}
					}
					break
				}
			}
			var pers [][]string
			var ks []string
			for k := range set {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			for _, k := range ks {
				pers = append(pers, []string{k, set[k]})
			}
			return pers, nil
		}
	}
}

func getInfoV2(host string) map[string]string {
	if !strings.HasPrefix(host, "http://") {
		host = "http://" + host
	}
	info := make(map[string]string)
	cli, ok := v2clients[host + "@" + rootClientName]
	if ok {
		ver, err := cli.GetVersion(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		memberKapi := client.NewMembersAPI(cli)
		member, err := memberKapi.Leader(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		info["version"] = ver.Server
		info["name"] = member.Name
		info["size"] = "unknow" // FIXME: How get?
	}
	return info
}

// v3 api
func connect(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	sess := sessmgr.SessionStart(w, r)
	host := r.FormValue("host")
	uname := r.FormValue("uname")
	passwd := r.FormValue("passwd")
	var info map[string]string
	if *useAuth {
		_, ok := clients[host + "@" + rootClientName]
		if !ok && uname != "root" {
			b, _ := json.Marshal(map[string]interface{}{"status":"root"})
			io.WriteString(w, string(b))
			return
		}
		if uname == "" || passwd == "" {
			b, _ := json.Marshal(map[string]interface{}{"status":"login"})
			io.WriteString(w, string(b))
			return
		}
	}
	if v := sess.Get("client"); v != nil {
		c := v.(*clientv3.Client)
		if uname == c.Username {
			if host == c.Endpoints()[0] {
				info = getInfo(host)
				b, _ := json.Marshal(map[string]interface{}{"status":"running", "info":info})
				io.WriteString(w, string(b))
				return
			}
		}
	} else {
		if c, ok := clients[host + "@" + uname]; ok {
			c.Close()
		}
	}
	endpoints := []string{host}
	var err error

	// use tls if usetls is true
	var tlsConfig *tls.Config
	if *usetls {
		tlsInfo := transport.TLSInfo{
			CertFile:      *cert,
			KeyFile:       *keyfile,
			TrustedCAFile: *cacert,
		}
		tlsConfig, err = tlsInfo.ClientConfig()
		if err != nil {
			log.Println(err.Error())
		}
	}

	conf := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		TLS:         tlsConfig,
	}
	if *useAuth {
		conf.Username = uname
		conf.Password = passwd
	}

	var c *clientv3.Client
	c, err = clientv3.New(conf)
	if err != nil {
		log.Println(r.Method, "v3", "connect fail.")
		b, _ := json.Marshal(map[string]interface{}{"status":"error", "message":err.Error()})
		io.WriteString(w, string(b))
	} else {
		clients[host + "@" + uname] = c
		if *useAuth {
			if uname == "root" { // etcd must be root to manage
				clients[host + "@" + rootClientName] = c
			}
		} else {
			clients[host + "@" + rootClientName] = c
		}
		sess.Set("client", c)
		sess.Set("clientName", host + "@" + uname)
		log.Println(r.Method, "v3", "connect success.")
		info = getInfo(host)
		b, _ := json.Marshal(map[string]interface{}{"status":"ok", "info":info})
		io.WriteString(w, string(b))
	}
}

func put(w http.ResponseWriter, r *http.Request) {
	cli := getClient(w, r)
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
		_, err = cli.Put(context.Background(), key, value, clientv3.WithLease(leaseResp.ID))
	} else {
		_, err = cli.Put(context.Background(), key, value)
	}
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if resp, err := cli.Get(context.Background(), key);err != nil {
			data["errorCode"] = 500
			data["errorCode"] = err.Error()
		} else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = getTTL(cli, kv.Lease)
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				data["node"] = node
			}
		}
	}

	var dataByte []byte
	if dataByte, err = json.Marshal(data);err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func get(w http.ResponseWriter, r *http.Request) {
	cli := getClient(w, r)
	key := r.FormValue("key")
	data := make(map[string]interface{})
	log.Println("GET", "v3", key)

	permissions, e := getPermissionPrefix(getClientName(w, r), key)
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
				resp, err = cli.Get(context.Background(), p[0], clientv3.WithPrefix())
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
						node["ttl"] = getTTL(cli, kv.Lease)
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
		if resp, err := cli.Get(context.Background(), key);err != nil {
			data["errorCode"] = 500
			data["message"] = err.Error()
		} else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = getTTL(cli, kv.Lease)
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				data["node"] = node
			} else {
				data["errorCode"] = 500
				data["message"] = "The node does not exist."
			}
		}
	}

	var dataByte []byte
	var err error
	if dataByte, err = json.Marshal(data);err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func getPath(w http.ResponseWriter, r *http.Request) {
	cli := getClient(w, r)
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
	permissions, e := getPermissionPrefix(getClientName(w, r), originKey)
	if e != nil {
		io.WriteString(w, e.Error())
		return
	}

	// parent
	var (
		presp *clientv3.GetResponse
		err   error
	)
	if originKey != separator {
		presp, err = cli.Get(context.Background(), originKey)
		if err != nil {
			data["errorCode"] = 500
			data["message"] = err.Error()
			dataByte, _ := json.Marshal(data)
			io.WriteString(w, string(dataByte))
			return
		}
	}
	if originKey == separator {
		min = 1
		//prefixKey = separator
	} else {
		min = len(strings.Split(originKey, separator))
		//prefixKey = originKey
	}
	max = min
	all[min] = []map[string]interface{}{{"key":originKey}}
	if presp != nil && presp.Count != 0 {
		all[min][0]["value"] = string(presp.Kvs[0].Value)
		all[min][0]["ttl"] = getTTL(cli, presp.Kvs[0].Lease)
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
			if string(kv.Key) == separator {
				continue
			}
			keys := strings.Split(string(kv.Key), separator) // /foo/bar
			for i := range keys { // ["", "foo", "bar"]
				k := strings.Join(keys[0:i+1], separator)
				if k == "" {
					continue
				}
				node := map[string]interface{}{"key":k}
				if node["key"].(string) == string(kv.Key) {
					node["value"] = string(kv.Value)
					if key == string(kv.Key) {
						node["ttl"] = getTTL(cli, kv.Lease)
					} else {
						node["ttl"] = 0
					}
					node["createdIndex"] = kv.CreateRevision
					node["modifiedIndex"] = kv.ModRevision
				}
				level := len(strings.Split(k, separator))
				if level > max {
					max = level
				}

				if _, ok := all[level];!ok {
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
					if strings.HasPrefix(a["key"].(string), pa["key"].(string) +separator) {
						pa["nodes"] = append(pa["nodes"].([]map[string]interface{}), a)
						pa["dir"] = true
					}
				}
			}
		}
	}
	data = all[min][0]
	if dataByte, err := json.Marshal(map[string]interface{}{"node":data});err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func del(w http.ResponseWriter, r *http.Request) {
	cli := getClient(w, r)
	key := r.FormValue("key")
	dir := r.FormValue("dir")
	log.Println("DELETE", "v3", key)

	if _, err := cli.Delete(context.Background(), key);err != nil {
		io.WriteString(w, err.Error())
		return
	}

	if dir == "true" {
		if _, err := cli.Delete(context.Background(), key +separator, clientv3.WithPrefix());err != nil {
			io.WriteString(w, err.Error())
			return
		}
	}
	io.WriteString(w, "ok")
}

func getTTL(cli *clientv3.Client, lease int64) int64 {
	resp, err := cli.Lease.TimeToLive(context.Background(), clientv3.LeaseID(lease))
	if err != nil {
		return 0
	}
	if resp.TTL == -1 {
		return 0
	}
	return resp.TTL
}

func getSeparator(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, separator)
}

func getClient(w http.ResponseWriter, r *http.Request) *clientv3.Client {
	sess := sessmgr.SessionStart(w, r)
	v := sess.Get("client")
	if v != nil {
		return v.(*clientv3.Client)
	}
	return nil
}

func getClientName(w http.ResponseWriter, r *http.Request) string {
	sess := sessmgr.SessionStart(w, r)
	v := sess.Get("clientName")
	if v != nil {
		return v.(string)
	}
	return ""
}

func getPermissionPrefix(clientName, key string) ([][]string, error) {
	if !*useAuth {
		return [][]string{{key, "p"}}, nil // No auth return all
	} else {
		a := strings.Split(clientName, "@")
		host, uname := a[0], a[1]

		if uname == "root" {
			return [][]string{{key, "p"}}, nil
		}

		rootCli := clients[host + "@" + rootClientName]

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

func getInfo(host string) map[string]string {
	info := make(map[string]string)
	cli, ok := clients[host + "@" + rootClientName]
	if ok {
		status, err := cli.Status(context.Background(), host)
		if err != nil {
			log.Fatal(err)
		}
		mems, err := cli.MemberList(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		kb := 1024
		mb := kb*1024
		gb := mb*1024
		var sizeStr string
		for _, m := range mems.Members {
			if m.ID == status.Leader {
				info["version"] = status.Version
				gn, rem1 := size(m.Size(), gb)
				mn, rem2 := size(rem1, mb)
				kn, bn := size(rem2, kb)
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
	}
	return info
}

func size(num int, unit int) (n, rem int) {
	return num/unit, num - (num/unit)*unit
}