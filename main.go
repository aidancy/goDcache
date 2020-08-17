package main

/*
$ curl http://localhost:9999/_geecache/scores/Tom
630
$ curl http://localhost:9999/_geecache/scores/kkk
kkk not exist
*/

import (
	"flag"
	"fmt"
	"gocache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *gocache.Group {
	return gocache.NewGroup("scores", 2<<10, gocache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

//启动缓存服务器
func statrCacheServer(addr string, addrs []string, gee *gocache.Group) {
	//创建 HTTPPool
	peers := gocache.NewHTTPPool(addr)
	//添加节点信息
	peers.Set(addrs...)
	//注册到 gee 中
	gee.RegisterPeers(peers)
	log.Println("gocache is running at", addr)
	//启动 HTTP 服务（共3个端口，8001/8002/8003），用户不感知。
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

//启动一个 API 服务（端口 9999）
func startAPIServer(apiAddr string, gee *gocache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fonted server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Gocache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string {
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	if api {
		go startAPIServer(apiAddr, gee)
	}
	statrCacheServer(addrMap[port], []string(addrs), gee)
}
