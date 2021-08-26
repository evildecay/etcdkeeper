package etcdkeeper

import (
	"etcdkeeper/session"
	_ "etcdkeeper/session/providers/memory"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type userInfo struct {
	host   string
	uname  string
	passwd string
}

type Etcdkeeper struct {
	rootUsers   map[string]*userInfo // host:rootUser
	rootUsersV2 map[string]*userInfo // host:rootUser

	config  EtcdkeeperConfig
	sessmgr *session.Manager
	mu      sync.Mutex
}

// NewEtcdKeeper initialize etdkeeper data structure
func NewEtcdKeeper(config *EtcdkeeperConfig) *Etcdkeeper {

	// Session managment
	sessmgr, err := session.NewManager("memory", "_etcdkeeper_session", 86400)
	if err != nil {
		log.Fatal(err)
	}
	time.AfterFunc(86400*time.Second, func() {
		sessmgr.GC()
	})

	return &Etcdkeeper{
		rootUsers:   make(map[string]*userInfo),
		rootUsersV2: make(map[string]*userInfo),

		config:  *config,
		sessmgr: sessmgr,
	}
}

func (ek *Etcdkeeper) GetSeparator(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, ek.config.separator)
}

func (ek *Etcdkeeper) size(num int, unit int) (n, rem int) {
	return num / unit, num - (num/unit)*unit
}
