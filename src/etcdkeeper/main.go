package main

import (
	"embed"
	"etcdkeeper/internal"
	"etcdkeeper/internal/etcdkeeper"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
)

//go:embed assets/*
var assets embed.FS

func main() {
	etcdkeeperConfig := &etcdkeeper.EtcdkeeperConfig{}
	etcdkeeperConfig.ParseFlag()

	host := flag.String("h", "0.0.0.0", "etcdkeeper listen hostname or ip address")
	port := flag.Int("p", 8080, "etcdkeeper listen port")

	flag.CommandLine.Parse(os.Args[1:])

	etcdk := etcdkeeper.NewEtcdKeeper(etcdkeeperConfig)

	middleware := func(fns ...func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			// avoid calling superfluous write on ResponseWriter
			cw := internal.NewCompletableResponseWriter(w)

			for _, fn := range fns {
				if cw.IsCompleted() {
					break
				}
				fn(cw, r)
			}
		}
	}

	// v2
	//http.HandleFunc(*name, v2request)
	http.HandleFunc("/v2/separator", middleware(nothing, etcdk.GetSeparator))
	http.HandleFunc("/v2/connect", middleware(nothing, etcdk.ConnectV2))
	http.HandleFunc("/v2/put", middleware(nothing, etcdk.PutV2))
	http.HandleFunc("/v2/get", middleware(nothing, etcdk.GetV2))
	http.HandleFunc("/v2/delete", middleware(nothing, etcdk.DelV2))
	// dirctory mode
	http.HandleFunc("/v2/getpath", middleware(nothing, etcdk.GetPathV2))

	// v3
	http.HandleFunc("/v3/separator", middleware(nothing, etcdk.GetSeparator))
	http.HandleFunc("/v3/connect", middleware(nothing, etcdk.Connect))
	http.HandleFunc("/v3/put", middleware(nothing, etcdk.Put))
	http.HandleFunc("/v3/get", middleware(nothing, etcdk.Get))
	http.HandleFunc("/v3/delete", middleware(nothing, etcdk.Del))
	// dirctory mode
	http.HandleFunc("/v3/getpath", middleware(nothing, etcdk.GetPath))

	// static directory server
	staticFS, err := fs.Sub(assets, "assets/static")
	if err != nil {
		log.Fatalf("Fail to load static assets resource directory : %v", err)
	}
	staticHandler := http.FileServer(http.FS(staticFS))
	templateFS, err := fs.Sub(assets, "assets/templates")
	if err != nil {
		log.Fatalf("Fail to load templates assets resource directory : %v", err)
	}
	templateHandler := internal.NewTemplateServer(templateFS, etcdkeeperConfig)

	http.HandleFunc("/", middleware(templateHandler.ServeHTTP, staticHandler.ServeHTTP))

	// listening
	log.Printf("listening on %s:%d\n", *host, *port)
	err = http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func nothing(_ http.ResponseWriter, _ *http.Request) {
	// Nothing
}
