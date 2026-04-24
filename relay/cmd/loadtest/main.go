// Command loadtest drives N concurrent WebSocket clients against the relay,
// measures connect time, end-to-end broadcast latency, and message drops.
//
// Usage:
//
//	go run ./cmd/loadtest -base=http://localhost:8080 -n=500 -m=10 -rate=50
//
// -n      total connections (all share the same user_id, so all broadcasts
//         go to everyone; a worst-case fanout scenario)
// -m      messages each client publishes
// -rate   target publishes per second, aggregated across all clients
// -warmup seconds to hold connections open after connect before publishing
//
// Tip: raise your file descriptor limit before scaling up:
//
//	ulimit -n 65536
//
// The relay itself needs the same treatment; the docker-compose.yml sets it.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	baseFlag   = flag.String("base", "http://localhost:8080", "relay HTTP base URL")
	nFlag      = flag.Int("n", 100, "number of concurrent WebSocket clients")
	mFlag      = flag.Int("m", 10, "messages each client publishes")
	rateFlag   = flag.Int("rate", 100, "aggregate publishes per second across all clients")
	warmupFlag = flag.Duration("warmup", 2*time.Second, "hold after connect before publishing")
	timeout    = flag.Duration("timeout", 60*time.Second, "overall test timeout")
)

type stats struct {
	connectMicros   []int64
	deliveredMicros []int64
	connected       int64
	published       int64
	delivered       int64
	dropped         int64
}

func main() {
	flag.Parse()

	tok, userID := register(*baseFlag)
	log.Printf("registered: user_id=%s", userID)

	wsURL := toWS(*baseFlag) + "/ws?token=" + url.QueryEscape(tok)

	s := &stats{}
	var wg sync.WaitGroup

	// Connect phase.
	clients := make([]*websocket.Conn, *nFlag)
	for i := 0; i < *nFlag; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := time.Now()
			c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				log.Printf("client %d dial: %v", i, err)
				return
			}
			clients[i] = c
			atomic.AddInt64(&s.connected, 1)
			dur := time.Since(start)
			s.connectMicros = append(s.connectMicros, dur.Microseconds())
		}(i)
	}
	wg.Wait()
	log.Printf("connected: %d / %d", s.connected, *nFlag)
	if s.connected == 0 {
		log.Fatal("no clients connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Reader goroutines: each client counts received broadcasts.  Broadcasts
	// carry a publish timestamp so we can compute end-to-end latency.
	var deliveredMu sync.Mutex
	for i, c := range clients {
		if c == nil {
			continue
		}
		go func(i int, c *websocket.Conn) {
			for {
				_, data, err := c.ReadMessage()
				if err != nil {
					return
				}
				// Parse just enough to extract the publish timestamp in `ts`.
				var m struct {
					TS int64 `json:"ts"`
				}
				_ = json.Unmarshal(data, &m)
				now := time.Now().UnixMicro()
				atomic.AddInt64(&s.delivered, 1)
				if m.TS > 0 {
					deliveredMu.Lock()
					s.deliveredMicros = append(s.deliveredMicros, now-m.TS)
					deliveredMu.Unlock()
				}
			}
		}(i, c)
	}

	time.Sleep(*warmupFlag)
	log.Printf("warmup done — starting publishes")

	// Publish phase: staggered across all clients to match target rate.
	totalMsgs := int64(*nFlag) * int64(*mFlag)
	interval := time.Second / time.Duration(*rateFlag)
	if interval < time.Microsecond {
		interval = time.Microsecond
	}
	start := time.Now()
	for j := 0; j < *mFlag; j++ {
		for i, c := range clients {
			if c == nil {
				continue
			}
			select {
			case <-ctx.Done():
				goto done
			default:
			}
			tsMicro := time.Now().UnixMicro()
			body := fmt.Sprintf(
				`{"device_id":"loadtest-%d","user_id":%q,"namespace":%q,"role":"provider","op":"upsert","table":"records","payload":{"id":"lt-%d-%d"},"ts":%d}`,
				i, userID, userID, i, j, tsMicro,
			)
			if err := c.WriteMessage(websocket.TextMessage, []byte(body)); err != nil {
				atomic.AddInt64(&s.dropped, 1)
				continue
			}
			atomic.AddInt64(&s.published, 1)
			time.Sleep(interval)
		}
	}
done:
	publishElapsed := time.Since(start)

	// Drain: wait up to timeout for delivered to reach published*connected,
	// or flatline for 1s.
	expected := atomic.LoadInt64(&s.published) * s.connected
	var last int64
	stableSince := time.Now()
	for time.Since(start) < *timeout {
		cur := atomic.LoadInt64(&s.delivered)
		if cur >= expected {
			break
		}
		if cur == last {
			if time.Since(stableSince) > 2*time.Second {
				break
			}
		} else {
			last = cur
			stableSince = time.Now()
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Close connections.
	for _, c := range clients {
		if c != nil {
			_ = c.Close()
		}
	}

	report(s, publishElapsed, totalMsgs, expected)
}

func register(base string) (token, userID string) {
	body := `{"device_id":"loadtest","platform":"browser"}`
	resp, err := http.Post(base+"/auth/register", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		log.Fatalf("register: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out struct {
		Token  string `json:"token"`
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		log.Fatalf("register decode: %v", err)
	}
	return out.Token, out.UserID
}

func toWS(base string) string {
	if strings.HasPrefix(base, "https") {
		return "wss" + base[5:]
	}
	if strings.HasPrefix(base, "http") {
		return "ws" + base[4:]
	}
	return base
}

func report(s *stats, publishElapsed time.Duration, totalMsgs, expected int64) {
	sort.Slice(s.connectMicros, func(i, j int) bool { return s.connectMicros[i] < s.connectMicros[j] })
	sort.Slice(s.deliveredMicros, func(i, j int) bool { return s.deliveredMicros[i] < s.deliveredMicros[j] })

	pub := atomic.LoadInt64(&s.published)
	del := atomic.LoadInt64(&s.delivered)
	drp := atomic.LoadInt64(&s.dropped)

	fmt.Fprintf(os.Stdout, "\n========== appunvs relay load test ==========\n")
	fmt.Fprintf(os.Stdout, "target:        %s\n", *baseFlag)
	fmt.Fprintf(os.Stdout, "connections:   %d (wanted %d)\n", s.connected, *nFlag)
	fmt.Fprintf(os.Stdout, "publishes:     %d (wanted %d, dropped %d)\n", pub, totalMsgs, drp)
	fmt.Fprintf(os.Stdout, "deliveries:    %d (expected %d, gap %d)\n", del, expected, expected-del)
	fmt.Fprintf(os.Stdout, "publish time:  %s\n", publishElapsed)
	if pub > 0 {
		fmt.Fprintf(os.Stdout, "publish rate:  %.1f/s\n", float64(pub)/publishElapsed.Seconds())
	}
	if del > 0 {
		fmt.Fprintf(os.Stdout, "fanout rate:   %.1f/s\n", float64(del)/publishElapsed.Seconds())
	}
	fmt.Fprintf(os.Stdout, "\nconnect (ms):  p50=%.1f p95=%.1f p99=%.1f\n",
		pct(s.connectMicros, 50)/1000, pct(s.connectMicros, 95)/1000, pct(s.connectMicros, 99)/1000)
	if len(s.deliveredMicros) > 0 {
		fmt.Fprintf(os.Stdout, "e2e lat (ms):  p50=%.1f p95=%.1f p99=%.1f max=%.1f\n",
			pct(s.deliveredMicros, 50)/1000, pct(s.deliveredMicros, 95)/1000,
			pct(s.deliveredMicros, 99)/1000, float64(s.deliveredMicros[len(s.deliveredMicros)-1])/1000)
	}
}

func pct(sorted []int64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := (len(sorted) - 1) * p / 100
	return float64(sorted[i])
}
