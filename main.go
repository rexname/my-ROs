package main

import (
    "encoding/json"
    "log"
    "math/rand"
    "net/http"
    "os"
    "os/exec"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

type BlockRequest struct {
    App string `json:"app"`
    On  bool   `json:"on"`
}

var allowedApps = map[string]struct{}{
    "youtube": {},
    "tiktok":  {},
    "spotify": {},
}

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin:     func(r *http.Request) bool { return true },
}

type hub struct {
    mu      sync.Mutex
    clients map[*websocket.Conn]struct{}
}

func newHub() *hub {
    return &hub{clients: make(map[*websocket.Conn]struct{})}
}

func (h *hub) add(c *websocket.Conn) {
    h.mu.Lock()
    h.clients[c] = struct{}{}
    h.mu.Unlock()
}

func (h *hub) remove(c *websocket.Conn) {
    h.mu.Lock()
    delete(h.clients, c)
    h.mu.Unlock()
}

func (h *hub) broadcast(v interface{}) {
    h.mu.Lock()
    for c := range h.clients {
        c.SetWriteDeadline(time.Now().Add(2 * time.Second))
        if err := c.WriteJSON(v); err != nil {
            c.Close()
            delete(h.clients, c)
        }
    }
    h.mu.Unlock()
}

func handleBlock(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    var br BlockRequest
    if err := json.NewDecoder(r.Body).Decode(&br); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
    br.App = strings.ToLower(br.App)
    if _, ok := allowedApps[br.App]; !ok {
        http.Error(w, "unknown app", http.StatusBadRequest)
        return
    }

    setName := "block_" + br.App
    var cmd *exec.Cmd
    if br.On {
        cmd = exec.Command("nft", "add", "element", "inet", "myapp", setName, "{ 0.0.0.0 . 0 timeout 300s }")
    } else {
        cmd = exec.Command("nft", "delete", "element", "inet", "myapp", setName, "{ 0.0.0.0 . 0 }")
    }
    out, err := cmd.CombinedOutput()
    if err != nil {
        log.Printf("nft cmd error: %v, output: %s", err, string(out))
        w.WriteHeader(http.StatusInternalServerError)
        _ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": err.Error(), "output": string(out)})
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

func handleWS(h *hub) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        c, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            return
        }
        h.add(c)
        defer h.remove(c)
        c.SetReadLimit(1024)
        for {
            if _, _, err := c.ReadMessage(); err != nil {
                return
            }
        }
    }
}

func main() {
    rand.Seed(time.Now().UnixNano())

    h := newHub()

    http.HandleFunc("/api/block", handleBlock)
    http.HandleFunc("/ws", handleWS(h))
    staticPath := "www"
    if _, err := os.Stat("/www"); err == nil {
        staticPath = "/www"
    }
    http.Handle("/", http.FileServer(http.Dir(staticPath)))

    go func() {
        apps := []string{"youtube", "tiktok", "spotify"}
        ips := []string{"142.250.1.1", "104.244.42.1", "34.98.99.1"}
        ports := []int{80, 443}
        for {
            app := apps[rand.Intn(len(apps))]
            ip := ips[rand.Intn(len(ips))]
            port := ports[rand.Intn(len(ports))]
            h.broadcast(map[string]interface{}{
                "type": "flow",
                "txt":  app + " " + ip + ":" + strconv.Itoa(port),
            })
            time.Sleep(2 * time.Second)
        }
    }()

    port := ":80"
    if p := os.Getenv("PORT"); p != "" {
        if strings.HasPrefix(p, ":") {
            port = p
        } else {
            port = ":" + p
        }
    }
    log.Printf("listening on %s", port)
    if err := http.ListenAndServe(port, nil); err != nil {
        log.Fatal(err)
    }
}
