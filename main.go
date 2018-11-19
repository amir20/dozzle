package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	flag "github.com/spf13/pflag"
)

var (
	cli      *client.Client
	addr     = flag.String("addr", ":8080", "http service address")
	base     = flag.String("base", "/", "base address of the application to mount")
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
	r := mux.NewRouter()
	s := r.PathPrefix(*base).Subrouter()
	box := packr.NewBox("./static")

	s.HandleFunc("/api/containers.json", listContainers)
	s.HandleFunc("/api/logs", logs)
	s.HandleFunc("/version", versionHandler)
	s.PathPrefix("/").Handler(http.StripPrefix(*base, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fileServer := http.FileServer(box)
		if box.Has(req.URL.Path) && req.URL.Path != "" && req.URL.Path != "/" {
			fileServer.ServeHTTP(w, req)
		} else {
			handleIndex(box, w)
		}
	})))

	log.Fatal(http.ListenAndServe(*addr, r))
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

func handleIndex(box packr.Box, w http.ResponseWriter) {
	text, _ := box.FindString("index.html")
	text = strings.Replace(text, "__BASE__", "{{ .Base }}", -1)
	tmpl, err := template.New("index.html").Parse(text)
	if err != nil {
		panic(err)
	}

	path := ""
	if *base != "/" {
		path = *base
	}
	data := struct{ Base string }{Base: path}
	err = tmpl.Execute(w, data)
	if err != nil {
		panic(err)
	}
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
			log.Panicln(err)
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
