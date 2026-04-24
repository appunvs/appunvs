# appunvs relay

Go WebSocket relay for the appunvs data-sync system. Authenticates devices,
assigns monotonic per-namespace `seq` numbers via Redis `INCR`, persists
provider-origin messages to a Redis Stream (24h effective retention) for
offline catch-up, and broadcasts messages to connected peers.

This service holds no business state. All mutations are opaque JSON payloads.

## Prerequisites

- Go 1.24+
- Redis 7+ (local, docker, or remote)

## Run locally

```bash
# from repo root
cd relay
go mod tidy
go run ./cmd/server -config config/config.yaml
```

On first run with empty `auth.private_key_path`, the relay generates an
ephemeral RS256 keypair and logs a warning. Tokens will not survive restart
in that mode; mount real keys for production.

## Run with docker-compose

```bash
cd relay
docker-compose up --build
```

Brings up Redis and the relay on `:8080`.

## Configuration

`config/config.yaml` is the canonical config. Every key can be overridden by
environment variables with the `APPUNVS_` prefix; dots become underscores.

| Key | Env | Default | Notes |
| --- | --- | --- | --- |
| `listen` | `APPUNVS_LISTEN` | `:8080` | HTTP listen addr |
| `redis.addr` | `APPUNVS_REDIS_ADDR` | `localhost:6379` | |
| `redis.password` | `APPUNVS_REDIS_PASSWORD` | `""` | |
| `redis.db` | `APPUNVS_REDIS_DB` | `0` | |
| `auth.private_key_path` | `APPUNVS_AUTH_PRIVATE_KEY_PATH` | `""` | RSA PEM (PKCS1 or PKCS8). Empty -> ephemeral |
| `auth.public_key_path` | `APPUNVS_AUTH_PUBLIC_KEY_PATH` | `""` | Optional; derived from private if absent |
| `auth.issuer` | `APPUNVS_AUTH_ISSUER` | `appunvs-relay` | |
| `auth.audience` | `APPUNVS_AUTH_AUDIENCE` | `appunvs-clients` | |
| `auth.ttl_hours` | `APPUNVS_AUTH_TTL_HOURS` | `24` | |
| `stream.max_len` | `APPUNVS_STREAM_MAX_LEN` | `100000` | Approximate XADD MAXLEN |
| `log.level` | `APPUNVS_LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |

## HTTP / WS surface

### `GET /health`

```bash
curl -s localhost:8080/health
# -> ok
```

### `POST /auth/register`

```bash
curl -s -X POST localhost:8080/auth/register \
  -H 'content-type: application/json' \
  -d '{"device_id":"dev-1","platform":"desktop"}'
# -> {"token":"<RS256 JWT>","user_id":"<uuid>"}
```

Omit `device_id` to have the server mint one. `platform` accepts either
`browser|desktop|mobile` or the fully-qualified `PLATFORM_*` form.

### `GET /ws?token=<JWT>&last_seq=<int>`

```bash
TOKEN=$(curl -s -X POST localhost:8080/auth/register \
  -H 'content-type: application/json' \
  -d '{"device_id":"dev-1","platform":"desktop"}' | jq -r .token)

wscat -c "ws://localhost:8080/ws?token=$TOKEN&last_seq=0"
```

Once connected, send a Message (canonical protojson; `seq` is assigned by
the relay and must be omitted on provider writes):

```json
{"role":"provider","op":"upsert","table":"records","namespace":"<user_id>","payload":{"id":"r1","data":"hi"},"ts":1714000000000}
```

`last_seq=N` triggers a Redis Stream replay of every provider message with
`seq > N` before the live loop begins.

## Message flow

1. **Provider write:** client sends a Message with `role=provider` and no
   `seq`. Relay `INCR`s `seq:{ns}`, `XADD`s the full JSON onto
   `stream:{ns}` with `MAXLEN ~ 100000`, then broadcasts to every other
   socket in the namespace.
2. **Connector write:** client sends a Message with `role=connector`. Relay
   forwards it to providers only (no seq, no stream persistence).
3. **Catch-up:** on WS open with `last_seq>0`, every stream entry with
   `seq > last_seq` is written to the socket in order before live fanout.

## Wire types

Types in `internal/pb/` are hand-written to mirror
`shared/proto/appunvs.proto`. If the proto changes, update these by hand.
The JSON encoding is canonical protojson: snake_case field names, enum
values as short lowercase strings (`"provider"`, `"upsert"`).

## Layout

```
relay/
  cmd/server/main.go
  internal/
    pb/            hand-written types + protojson (de)coder
    auth/          RS256 signer + /auth/register
    sequencer/     INCR seq:{ns}
    stream/        XADD / XRANGE on stream:{ns}
    hub/           in-memory fanout
    handler/       /ws entrypoint
    config/        viper loader
  config/config.yaml
  Dockerfile
  docker-compose.yml
```

## Known scaffold stubs

- No user store: every `/auth/register` call mints a fresh `user_id`.
- No TLS: terminate in a reverse proxy.
- No metrics, no rate limiting.
- No tests: scaffold target is `go build ./... && go vet ./...`.
