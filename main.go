package main

import (
    "context"
    "encoding/binary"
    "encoding/json"
    "fmt"
    "github.com/amir20/dozzle/docker"
    "github.com/gobuffalo/packr"
    "github.com/gorilla/mux"
    flag "github.com/spf13/pflag"
    "html/template"
    "log"
    "net/http"
    "strings"
)

var (
    dockerClient docker.DockerClient
    addr         = ""
    base         = "/"
    version      = "dev"
    commit       = "none"
    date         = "unknown"
)

func init() {
    flag.StringVar(&addr, "addr", ":8080", "http service address")
    flag.StringVar(&base, "base", "/", "base address of the application to mount")

    dockerClient = docker.NewDockerClient()
    flag.Parse()
}

func main() {
    r := mux.NewRouter()

    if base != "/" {
        r.HandleFunc(base, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
            http.Redirect(w, req, base+"/", http.StatusMovedPermanently)
        }))
    }

    s := r.PathPrefix(base).Subrouter()
    box := packr.NewBox("./static")

    s.HandleFunc("/api/containers.json", listContainers)
    s.HandleFunc("/api/logs/stream", streamLogs)
    s.HandleFunc("/api/events/stream", streamEvents)
    s.HandleFunc("/version", versionHandler)
    s.PathPrefix("/").Handler(http.StripPrefix(base, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        fileServer := http.FileServer(box)
        if box.Has(req.URL.Path) && req.URL.Path != "" && req.URL.Path != "/" {
            fileServer.ServeHTTP(w, req)
        } else {
            handleIndex(box, w)
        }
    })))

    log.Printf("Accepting connections on %s", addr)
    log.Fatal(http.ListenAndServe(addr, r))
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, version)
    fmt.Fprintln(w, commit)
    fmt.Fprintln(w, date)
}

func listContainers(w http.ResponseWriter, r *http.Request) {
    containers, err := dockerClient.ListContainers()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    err = json.NewEncoder(w).Encode(containers)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func handleIndex(box packr.Box, w http.ResponseWriter) {
    text, _ := box.FindString("index.html")
    text = strings.Replace(text, "__BASE__", "{{ .Base }}", -1)
    tmpl, err := template.New("index.html").Parse(text)
    if err != nil {
        panic(err)
    }

    path := ""
    if base != "/" {
        path = base
    }

    data := struct{ Base string }{path}
    err = tmpl.Execute(w, data)
    if err != nil {
        panic(err)
    }
}

func streamLogs(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, "id is required", http.StatusBadRequest)
        return
    }

    f, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
        return
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    reader, err := dockerClient.ContainerLogs(ctx, id)
    if err != nil {
        log.Println(err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer reader.Close()

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Transfer-Encoding", "chunked")

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
        _, err = fmt.Fprintf(w, "data: %s\n\n", content[:n])
        if err != nil {
            log.Println(err)
            break
        }
        f.Flush()
    }
}

func streamEvents(w http.ResponseWriter, r *http.Request) {
    f, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Transfer-Encoding", "chunked")

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    messages, _ := dockerClient.Events(ctx)

    for message := range messages {
        switch message.Action {
        case "connect":
            fallthrough
        case "disconnect":
            fallthrough
        case "create":
            fallthrough
        case "destroy":
            _, err := fmt.Fprintf(w, "event: containers-changed\n")
            _, err = fmt.Fprintf(w, "data: %s\n\n", message.Action)

            if err != nil {
                log.Println(err)
                break
            }
            f.Flush()
        default:
            // Do nothing
        }
    }
}
