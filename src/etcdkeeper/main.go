package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"github.com/coreos/etcd/client"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	cli       *clientv3.Client // v3 client
	kapi      client.KeysAPI   // v2 client
	sep       = flag.String("sep", "/", "separator")
	separator = ""
	usetls    = flag.Bool("usetls", false, "use tls")
	cacert    = flag.String("cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle")
	cert      = flag.String("cert", "", "identify secure client using this TLS certificate file")
	keyfile   = flag.String("key", "", "identify secure client using this TLS key file")
)

func main() {
	host := flag.String("h","0.0.0.0","host name or ip address")
	port := flag.Int("p", 8080, "port")
	//name := flag.String("n", "/request", "request root name for etcdv2")

	flag.CommandLine.Parse(os.Args[1:])
	separator = *sep

	// v2
	//http.HandleFunc(*name, v2request)
	http.HandleFunc("/v2/separator", getSeparator)
	http.HandleFunc("/v2/connect", connectV2)
	http.HandleFunc("/v2/put", putV2)
	http.HandleFunc("/v2/get", getV2)
	http.HandleFunc("/v2/delete", delV2)
	// dirctory mode
	http.HandleFunc("/v2/getpath", getPathV2)

	// v3
	http.HandleFunc("/v3/separator", getSeparator)
	http.HandleFunc("/v3/connect", connect)
	http.HandleFunc("/v3/put", put)
	http.HandleFunc("/v3/get", get)
	http.HandleFunc("/v3/delete", del)
	// dirctory mode
	http.HandleFunc("/v3/getpath", getPath)

	wd, err := os.Getwd()
	if err != nil{
		log.Fatal(err)
	}

	//log.Println(http.Dir(wd + "/assets"))

	http.Handle("/", http.FileServer(http.Dir(wd + "/assets"))) // view static directory

	log.Printf("listening on %s:%d\n", *host, *port)
	err = http.ListenAndServe(*host + ":" + strconv.Itoa(*port), nil)
	if err != nil {
		log.Fatal(err)
	}
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
	host := strings.TrimSpace(r.FormValue("host"))
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}
	endpoints := []string{host}

	cfg := client.Config{
		Endpoints:               endpoints,
		HeaderTimeoutPerRequest: 5*time.Second,
		//Username:"test",
		//Password:"test",
	}

	c, err := client.New(cfg)
	if err != nil {
		log.Println(r.Method, "v2", "connect fail.")
		io.WriteString(w, string(err.Error()))
	} else {
		kapi = client.NewKeysAPI(c)
		log.Println(r.Method, "v2", "connect success.")
		io.WriteString(w, "ok")
	}
}

func putV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	value := r.FormValue("value")
	ttl := r.FormValue("ttl")
	dir := r.FormValue("dir")
	log.Println("PUT", "v2", key)

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

	if resp, err := kapi.Get(context.Background(), key, &client.GetOptions{Recursive:true, Sort:true}); err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if resp.Node == nil {
			data["errorCode"] = 500
			data["message"] = "The node does not exist."
		} else {
			data["node"] = getNode(resp.Node)
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

func getNode(node *client.Node) map[string]interface{} {
	nm := make(map[string]interface{})
	nm["key"] = node.Key
	nm["value"] = node.Value
	nm["dir"] = node.Dir
	nm["ttl"] = node.TTL
	nm["createdIndex"] = node.CreatedIndex
	nm["modifiedIndex"] = node.ModifiedIndex
	nm["nodes"] = make([]map[string]interface{}, 0)
	if len(node.Nodes) != 0 {
		for _, n := range node.Nodes {
			nm["nodes"] = append(nm["nodes"].([]map[string]interface{}), getNode(n))
		}
	}
	return nm
}

func delV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	dir := r.FormValue("dir")
	log.Println("DELETE", "v2", key)

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
	key := r.FormValue("key")
	log.Println("GET", "v2", key)
	getV2(w, r)
}

// v3 api
func connect(w http.ResponseWriter, r *http.Request) {
	if cli != nil {
		etcdHost := cli.Endpoints()[0]
		if r.FormValue("host") == etcdHost {
			io.WriteString(w, "running")
			return
		}else {
			if err := cli.Close();err != nil {
				log.Println(err.Error())
			}
		}
	}
	endpoints := []string{r.FormValue("host")}
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

	cli, err = clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		TLS:         tlsConfig,
	})

	if err != nil {
		log.Println(r.Method, "v3", "connect fail.")
		io.WriteString(w, string(err.Error()))
	} else {
		log.Println(r.Method, "v3", "connect success.")
		io.WriteString(w, "ok")
	}
}

func getSeparator(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, separator)
}

func put(w http.ResponseWriter, r *http.Request) {
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
		if resp, err := cli.Get(context.Background(), key, clientv3.WithPrefix());err != nil {
			data["errorCode"] = err.Error()
		} else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = getTTL(kv.Lease)
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
	key := r.FormValue("key")
	data := make(map[string]interface{})
	log.Println("GET", "v3", key)

	if resp, err := cli.Get(context.Background(), key, clientv3.WithPrefix());err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if r.FormValue("prefix") == "true" {
			pnode := make(map[string]interface{})
			pnode["key"] = key
			pnode["nodes"] = make([]map[string]interface{}, 0)
			for _, kv := range resp.Kvs {
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				if key == string(kv.Key) {
					node["ttl"] = getTTL(kv.Lease)
				} else {
					node["ttl"] = 0
				}
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				nodes := pnode["nodes"].([]map[string]interface{})
				pnode["nodes"] = append(nodes, node)
			}
			data["node"] = pnode
		} else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = getTTL(kv.Lease)
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
	key := r.FormValue("key")
	log.Println("GET", "v3", key)
	var (
		data = make(map[string]interface{})
		/*
			{1:["/"], 2:["/foo", "/foo2"], 3:["/foo/bar", "/foo2/bar"], 4:["/foo/bar/test"]}
		 */
		all = make(map[int][]map[string]interface{})
		min int
		max int
		prefixKey string
	)
	// parent
	presp, err := cli.Get(context.Background(), key)
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
		dataByte, _ := json.Marshal(data)
		io.WriteString(w, string(dataByte))
		return
	}
	if key == separator {
		min = 1
		prefixKey = separator
	} else {
		min = len(strings.Split(key, separator))
		prefixKey = key + separator
	}
	max = min
	all[min] = []map[string]interface{}{{"key":key}}
	if presp.Count != 0 {
		all[min][0]["value"] = string(presp.Kvs[0].Value)
		all[min][0]["ttl"] = getTTL(presp.Kvs[0].Lease)
		all[min][0]["createdIndex"] = presp.Kvs[0].CreateRevision
		all[min][0]["modifiedIndex"] = presp.Kvs[0].ModRevision
	}
	all[min][0]["nodes"] = make([]map[string]interface{}, 0)

	//child
	resp, err := cli.Get(context.Background(), prefixKey, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
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
		var begin bool
		for i := range keys { // ["", "foo", "bar"]
			k := strings.Join(keys[0:i+1], separator)
			if k == "" {
				continue
			}
			if key == separator {
				begin = true
			} else if k == key {
				begin = true
				continue
			}
			if begin {
				node := map[string]interface{}{"key":k}
				if node["key"].(string) == string(kv.Key) {
					node["value"] = string(kv.Value)
					if key == string(kv.Key) {
						node["ttl"] = getTTL(kv.Lease)
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

func getTTL(lease int64) int64 {
	resp, err := cli.Lease.TimeToLive(context.Background(), clientv3.LeaseID(lease))
	if err != nil {
		return 0
	}
	if resp.TTL == -1 {
		return 0
	}
	return resp.TTL
}
