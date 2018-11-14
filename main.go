package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/websocket"
)

var (
	cli      *client.Client
	addr     = flag.String("addr", ":8080", "http service address")
	upgrader = websocket.Upgrader{}
	version  = "dev"
	commit   = "none"
	date     = "unknown"
)

func init() {
	var err error
	cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}
	flag.Parse()
}

func main() {
	box := packr.NewBox("./static")
	http.HandleFunc("/api/containers.json", listContainers)
	http.HandleFunc("/api/logs", logs)
	http.HandleFunc("/version", versionHandler)
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fileServer := http.FileServer(box)
		if box.Has(req.URL.Path) {
			fileServer.ServeHTTP(w, req)
		} else {
			bytes, _ := box.Find("index.html")
			w.Write(bytes)
		}
	}))

	log.Fatal(http.ListenAndServe(*addr, nil))
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, version)
	fmt.Fprintln(w, commit)
	fmt.Fprintln(w, date)
}

func listContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	json.NewEncoder(w).Encode(containers)
}

func logs(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer c.Close()

	options := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: "300", Timestamps: true}
	reader, err := cli.ContainerLogs(context.Background(), id, options)
	defer reader.Close()
	if err != nil {
		log.Fatal(err)
	}

	hdr := make([]byte, 8)
	content := make([]byte, 1024, 1024*1024)
	for {
		_, err := reader.Read(hdr)
		if err != nil {
			panic(err)
		}
		count := binary.BigEndian.Uint32(hdr[4:])
		n, err := reader.Read(content[:count])
		if err != nil {
			log.Println(err)
			break
		}
		err = c.WriteMessage(websocket.TextMessage, content[:n])
		if err != nil {
			log.Println(err)
			break
		}
	}
}
