package main

import(
	"io"
	"log"
	"net/http"
	"os"
	"io/ioutil"
	"strings"
	"time"
	"flag"
	"strconv"
	"encoding/json"
	"github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

var cli *clientv3.Client

func main() {
	host := flag.String("h","0.0.0.0","host name or ip address")
	port := flag.Int("p", 8080, "port")
	name := flag.String("n", "/request", "request root name")
	flag.CommandLine.Parse(os.Args[1:])

	// v2
	http.HandleFunc(*name, v2request)

	// v3
	http.HandleFunc("/connect", connect)
	http.HandleFunc("/put", put)
	http.HandleFunc("/get", get)
	http.HandleFunc("/delete", del)

	wd, err := os.Getwd()
	if err != nil{
		log.Fatal(err)
	}

	log.Println(http.Dir(wd + "/assets"))

	http.Handle("/", http.FileServer(http.Dir(wd + "/assets"))) // view static directory

	log.Printf("listening on %s:%d\n", *host, *port)
	err = http.ListenAndServe(*host + ":" + strconv.Itoa(*port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func v2request(w http.ResponseWriter, r *http.Request){
	if err := r.ParseForm(); err != nil {
		log.Println(err.Error())
	}
	log.Println(r.Method, "v2", r.FormValue("url"), r.PostForm.Encode())

	body := strings.NewReader(r.PostForm.Encode())
	req, err := http.NewRequest(r.Method, r.Form.Get("url"), body)
	if err != nil {
		io.WriteString(w, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 10*time.Second} // important!!!
	resp, err := client.Do(req)
	if err != nil {
		io.WriteString(w, err.Error())
	}else {
		result, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			io.WriteString(w, "Get data failed: " + err.Error())
		}else {
			io.WriteString(w, string(result))
		}
	}
}

func connect(w http.ResponseWriter, r *http.Request){
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
	cli, err = clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Println(r.Method, "v3", "connect fail.")
		io.WriteString(w, string(err.Error()))
	}else {
		log.Println(r.Method, "v3", "connect success.")
		io.WriteString(w, "ok")
	}
}

func put(w http.ResponseWriter, r *http.Request){
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
	}else {
		_, err = cli.Put(context.Background(), key, value)
	}
	if err != nil {
		io.WriteString(w, string(err.Error()))
	}else {
		if resp, err := cli.Get(context.Background(), key, clientv3.WithPrefix());err != nil {
			data["errorCode"] = err.Error()
		}else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = 0 // FIXME: clientv3.0 not support?
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				data["node"] = node
			}
		}
		var dataByte []byte
		if dataByte, err = json.Marshal(data);err != nil {
			io.WriteString(w, err.Error())
		} else {
			io.WriteString(w, string(dataByte))
		}
	}
}

func get(w http.ResponseWriter, r *http.Request){
	key := r.FormValue("key")
	data := make(map[string]interface{})
	log.Println("GET", "v3", key)

	if resp, err := cli.Get(context.Background(), key, clientv3.WithPrefix());err != nil {
		data["errorCode"] = err.Error()
	}else {
		if r.FormValue("prefix") == "true" {
			pnode := make(map[string]interface{})
			pnode["key"] = key
			pnode["nodes"] = make([]map[string]interface{}, 0)
			for _, kv := range resp.Kvs {
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = 0 // FIXME: clientv3.0 not support?
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				nodes := pnode["nodes"].([]map[string]interface{})
				pnode["nodes"] = append(nodes, node)
			}
			data["node"] = pnode
		}else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = 0 // FIXME: clientv3.0 not support?
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				data["node"] = node
			}else {
				data["errorCode"] = "The node does not exist."
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

func del(w http.ResponseWriter, r *http.Request){
	key := r.FormValue("key")
	log.Println("DELETE", "v3", key)

	if _, err := cli.Delete(context.Background(), key);err != nil {
		io.WriteString(w, err.Error())
	}else {
		io.WriteString(w, "ok")
	}
}
