# appunvs relay load test

Single-binary harness for measuring the relay's WebSocket throughput, fanout
rate, and end-to-end delivery latency. All clients register as the same user,
so every publish fans out to every other client — a worst-case scenario.

## Usage

```
go run ./cmd/loadtest -base=http://localhost:8080 -n=500 -m=5 -rate=100
```

Flags:

| flag     | default | meaning |
|----------|---------|---------|
| `-base`  | `http://localhost:8080` | relay HTTP base URL |
| `-n`     | `100`   | concurrent WebSocket clients |
| `-m`     | `10`    | messages each client publishes |
| `-rate`  | `100`   | aggregate publishes/sec target |
| `-warmup`| `2s`    | wait after connect before publishing |
| `-timeout`| `60s`  | total test timeout |

Output includes publish rate, fanout rate, delivery gap, connect-time
percentiles, and end-to-end latency percentiles.

## Preparation

Bump file descriptor limits for the driver **and** the relay before scaling
past a few hundred connections:

```
ulimit -n 65536
```

The shipped `docker-compose.yml` sets `ulimits.nofile=65536` for the relay
service; running natively requires raising it yourself. The loadtest driver
also opens one descriptor per connection plus a few for HTTP.

## Measured ceilings on this branch

Environment: Ubuntu 24.04, Go 1.24, single-core test, relay + Redis on
localhost, ulimit 4096.

| scenario | publishes | drops | gap | p99 e2e | fanout/s |
|----------|-----------|-------|-----|---------|----------|
| n=100, m=5, rate=500  | 500  | 0 | 0 | 1.8 ms   | 36k |
| n=500, m=5, rate=100  | 2500 | 0 | 0 | 4.7 ms   | 46k |
| n=500, m=5, rate=500  | 2366 | 134 | 60k | 190 ms | 182k |
| n=1000, m=3, rate=2000 | 2797 | 203 | 1.1M | 8.4 s | 559k |

Takeaway: the relay sustains ~46k fanouts/sec with zero loss at 500 connections
and a modest publish rate. Beyond that, the per-connection send buffer (64)
fills up during bursts, causing the hub to evict slow consumers — visible as
the "gap" column. Lifting that ceiling means:

1. raise send buffer size (currently `sendBuffer = 64` in `internal/hub/conn.go`);
2. split broadcast fanout across multiple workers per namespace;
3. prepare message bytes once before the fanout loop (current code marshals
   inside `WriteLoop`).

Until those land, treat 10k concurrent connections as an upper bound that
needs horizontal scaling rather than a single-node ceiling.
