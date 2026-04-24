# auth — account and device model

This supersedes the "register auto-mints a fresh user_id" shortcut that lived
in the scaffold. The relay is now a multi-tenant SaaS: real user accounts,
persistent devices, multiple JWT flavors.

## Tables

```sql
CREATE TABLE users (
  id            TEXT PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE COLLATE NOCASE,
  password_hash TEXT NOT NULL,
  created_at    INTEGER NOT NULL
);

CREATE TABLE devices (
  id         TEXT PRIMARY KEY,            -- client-generated UUID
  user_id    TEXT NOT NULL REFERENCES users(id),
  platform   TEXT NOT NULL,               -- browser|desktop|mobile
  created_at INTEGER NOT NULL,
  last_seen  INTEGER
);
CREATE INDEX idx_devices_user ON devices(user_id);
```

## Endpoints

| method | path              | auth     | purpose |
|--------|-------------------|----------|---------|
| POST   | /auth/signup      | -        | create account, return session JWT |
| POST   | /auth/login       | -        | verify credentials, return session JWT |
| POST   | /auth/register    | session  | register a device for the logged-in user, return device JWT |
| GET    | /auth/me          | session  | current user + device list |

### Request / response shapes

```json
// POST /auth/signup  →  200
{ "email": "alice@example.com", "password": "hunter2" }
// response
{ "user_id": "u_...", "session_token": "<JWT>" }

// POST /auth/login → 200 (same shape as signup)

// POST /auth/register  (Authorization: Bearer <session_token>)
{ "device_id": "d_...", "platform": "browser" }
// response
{ "device_token": "<JWT>" }
```

## Two JWT flavors

- **Session JWT** — issued on signup/login, TTL 24h (configurable),
  carries `uid` only. Used for dashboard API and `/auth/register`.
- **Device JWT** — issued on /auth/register, TTL 30d, carries `uid` +
  `did`. Used on `/ws`.

A single `Signer` handles both; the token's `typ` claim distinguishes them.

## Password hashing

`golang.org/x/crypto/bcrypt` with cost 10. Email is lower-cased before storage
and compared case-insensitively.

## Migration from scaffold

The legacy `/auth/register` that minted anonymous users is removed. Existing
device tokens issued by the scaffolded relay will verify (same keypair) until
they expire, but new devices must go through signup → login → register.

## Persistence

SQLite (pure-Go `modernc.org/sqlite`). The relay opens a single file at
`$APPUNVS_DB_PATH` (default `data/relay.db`). A future migration moves the
account data to Postgres; Redis stays dedicated to the fanout stream.
