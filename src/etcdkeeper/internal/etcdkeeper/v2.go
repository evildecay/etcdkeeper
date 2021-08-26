package etcdkeeper

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/client"
)

// v2 api
func (ek *Etcdkeeper) ConnectV2(w http.ResponseWriter, r *http.Request) {
	ek.mu.Lock()
	defer ek.mu.Unlock()
	sess := ek.sessmgr.SessionStart(w, r)
	host := strings.TrimSpace(r.FormValue("host"))
	uname := r.FormValue("uname")
	passwd := r.FormValue("passwd")
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}

	if ek.config.useAuth {
		_, ok := ek.rootUsersV2[host]
		if !ok && uname != "root" {
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

	if uinfo, ok := sess.Get("uinfov2").(*userInfo); ok {
		if host == uinfo.host && uname == uinfo.uname && passwd == uinfo.passwd {
			info := ek.getInfoV2(host)
			b, _ := json.Marshal(map[string]interface{}{"status": "running", "info": info})
			io.WriteString(w, string(b))
			return
		}
	}

	uinfo := &userInfo{host: host, uname: uname, passwd: passwd}
	_, err := ek.newClientV2(uinfo)
	if err != nil {
		log.Println(r.Method, "v2", "connect fail.")
		b, _ := json.Marshal(map[string]interface{}{"status": "error", "message": err.Error()})
		io.WriteString(w, string(b))
		return
	}
	_ = sess.Set("uinfov2", uinfo)

	if ek.config.useAuth {
		if uname == "root" {
			ek.rootUsersV2[host] = uinfo
		}
	} else {
		ek.rootUsersV2[host] = uinfo
	}
	log.Println(r.Method, "v2", "connect success.")
	info := ek.getInfoV2(host)
	b, _ := json.Marshal(map[string]interface{}{"status": "running", "info": info})
	io.WriteString(w, string(b))
}

func (ek *Etcdkeeper) PutV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	value := r.FormValue("value")
	ttl := r.FormValue("ttl")
	dir := r.FormValue("dir")
	log.Println("PUT", "v2", key)

	kapi := client.NewKeysAPI(ek.getClientV2(w, r))

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
		_, err = kapi.Set(context.Background(), key, value, &client.SetOptions{TTL: time.Duration(sec) * time.Second, Dir: isDir})
	} else {
		_, err = kapi.Set(context.Background(), key, value, &client.SetOptions{Dir: isDir})
	}
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if resp, err := kapi.Get(context.Background(), key, &client.GetOptions{Recursive: true, Sort: true}); err != nil {
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
	if dataByte, err = json.Marshal(data); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func (ek *Etcdkeeper) GetV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	data := make(map[string]interface{})
	log.Println("GET", "v2", key)

	var cli client.Client
	sess := ek.sessmgr.SessionStart(w, r)
	v := sess.Get("uinfov2")
	var uinfo *userInfo
	if v != nil {
		uinfo = v.(*userInfo)
		cli, _ = ek.newClientV2(uinfo)
		kapi := client.NewKeysAPI(cli)

		var permissions [][]string
		if r.FormValue("prefix") == "true" {
			var e error
			permissions, e = ek.getPermissionPrefixV2(uinfo.host, uinfo.uname, key)
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
		if key == ek.config.separator {
			min = 1
		} else {
			min = len(strings.Split(key, ek.config.separator))
		}
		max = min
		all := make(map[int][]map[string]interface{})
		if key == ek.config.separator {
			all[min] = []map[string]interface{}{{"key": key, "value": "", "dir": true, "nodes": make([]map[string]interface{}, 0)}}
		}
		for _, p := range permissions {
			pKey, pRange := p[0], p[1]
			var opt *client.GetOptions
			if pRange != "" {
				if pRange == "c" {
					pKey += ek.config.separator
				}
				opt = &client.GetOptions{Recursive: true, Sort: true}
			}
			if resp, err := kapi.Get(context.Background(), pKey, opt); err != nil {
				data["errorCode"] = 500
				data["message"] = err.Error()
			} else {
				if resp.Node == nil {
					data["errorCode"] = 500
					data["message"] = "The node does not exist."
				} else {
					max = ek.getNode(resp.Node, key, all, min, max)
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
						if strings.HasPrefix(a["key"].(string), pa["key"].(string)+ek.config.separator) {
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
	}

	var dataByte []byte
	var err error
	if dataByte, err = json.Marshal(data); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func nodesSort(node map[string]interface{}) {
	if v, ok := node["nodes"]; ok && v != nil {
		a := v.([]map[string]interface{})
		if len(a) != 0 {
			for i := 0; i < len(a)-1; i++ {
				nodesSort(a[i])
				for j := i + 1; j < len(a); j++ {
					if a[j]["key"].(string) < a[i]["key"].(string) {
						a[i], a[j] = a[j], a[i]
					}
				}
			}
			nodesSort(a[len(a)-1])
		}
	}
}

func (ek *Etcdkeeper) DelV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	dir := r.FormValue("dir")
	log.Println("DELETE", "v2", key)

	kapi := client.NewKeysAPI(ek.getClientV2(w, r))

	isDir, _ := strconv.ParseBool(dir)
	if isDir {
		if _, err := kapi.Delete(context.Background(), key, &client.DeleteOptions{Recursive: true, Dir: true}); err != nil {
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

func (ek *Etcdkeeper) GetPathV2(w http.ResponseWriter, r *http.Request) {
	ek.GetV2(w, r)
}

func (ek *Etcdkeeper) getClientV2(w http.ResponseWriter, r *http.Request) client.Client {
	sess := ek.sessmgr.SessionStart(w, r)
	v := sess.Get("uinfov2")
	if v != nil {
		uinfo := v.(*userInfo)
		c, _ := ek.newClientV2(uinfo)
		return c
	}
	return nil
}

func (ek *Etcdkeeper) newClientV2(uinfo *userInfo) (client.Client, error) {
	cfg := client.Config{
		Endpoints:               []string{uinfo.host},
		HeaderTimeoutPerRequest: time.Second * time.Duration(ek.config.connectTimeout),
	}
	if ek.config.useAuth {
		cfg.Username = uinfo.uname
		cfg.Password = uinfo.passwd
	}

	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (ek *Etcdkeeper) getPermissionPrefixV2(host, uname, key string) ([][]string, error) {
	if !ek.config.useAuth {
		return [][]string{{key, "p"}}, nil // No auth return all
	} else {
		if uname == "root" {
			return [][]string{{key, "p"}}, nil
		}

		if !strings.HasPrefix(host, "http://") {
			host = "http://" + host
		}
		rootUser := ek.rootUsersV2[host]
		rootCli, err := ek.newClientV2(rootUser)
		if err != nil {
			return nil, err
		}
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
								k = ks[:len(ks)-1]
								set[k] = "p"
							} else if strings.HasSuffix(ks, "/*") {
								k = ks[:len(ks)-2]
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

func (ek *Etcdkeeper) getInfoV2(host string) map[string]string {
	if !strings.HasPrefix(host, "http://") {
		host = "http://" + host
	}
	info := make(map[string]string)
	uinfo, ok := ek.rootUsersV2[host]
	if ok {
		rootClient, err := ek.newClientV2(uinfo)
		if err != nil {
			log.Println(err)
			return info
		}
		ver, err := rootClient.GetVersion(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		memberKapi := client.NewMembersAPI(rootClient)
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

func (ek *Etcdkeeper) getNode(node *client.Node, selKey string, all map[int][]map[string]interface{}, min, max int) int {
	keys := strings.Split(node.Key, ek.config.separator) // /foo/bar
	if len(keys) < min && strings.HasPrefix(node.Key, selKey) {
		return max
	}
	for i := range keys { // ["", "foo", "bar"]
		k := strings.Join(keys[0:i+1], ek.config.separator)
		if k == "" {
			continue
		}
		nodeMap := map[string]interface{}{"key": k, "dir": true, "nodes": make([]map[string]interface{}, 0)}
		if k == node.Key {
			nodeMap["value"] = node.Value
			nodeMap["dir"] = node.Dir
			nodeMap["ttl"] = node.TTL
			nodeMap["createdIndex"] = node.CreatedIndex
			nodeMap["modifiedIndex"] = node.ModifiedIndex
		}
		keylevel := len(strings.Split(k, ek.config.separator))
		if keylevel > max {
			max = keylevel
		}

		if _, ok := all[keylevel]; !ok {
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
			max = ek.getNode(n, selKey, all, min, max)
		}
	}
	return max
}
