package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gobuffalo/packr"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var (
	box = packr.NewBox("./templates")
	cli *client.Client
)

func init() {
	var err error
	cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	flag.Parse()
	// http.HandleFunc("/echo", echo)
	box := packr.NewBox("./dist")
	http.Handle("/", http.FileServer(box))

	http.HandleFunc("/contains.json", listContainers)

	log.Fatal(http.ListenAndServe(*addr, nil))

}

func listContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	json.NewEncoder(w).Encode(containers)
}
