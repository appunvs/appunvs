# appunvs mobile

Flutter app that participates in the **appunvs** sync mesh as a
**provider** only. It owns the authoritative local SQLite data (via
[drift]) and round-trips changes through the Go relay over a WebSocket.

## Prerequisites

- Flutter SDK `>=3.16` (stable channel recommended)
- A running appunvs relay reachable from the device / emulator
- Android Studio / Xcode for native toolchains

This scaffold **does not** include the native `android/` or `ios/`
folders. Generate them once with:

```bash
cd mobile
flutter create .
```

`flutter create .` will fill in the platform scaffolding without
overwriting any existing Dart source, `pubspec.yaml`, or
`analysis_options.yaml`.

## Install / codegen

```bash
flutter pub get
dart run build_runner build --delete-conflicting-outputs
```

The codegen step produces `lib/core/database/app_db.g.dart` and
`lib/core/database/records_dao.g.dart`. They are not checked in.

## Run

```bash
flutter run --dart-define=RELAY_BASE=http://10.0.2.2:8080
```

`10.0.2.2` is the host-loopback alias on the Android emulator. On iOS
simulator use `http://127.0.0.1:8080`. For a LAN device, use the host
machine's LAN IP.

### Configuration

| Key          | Default                   | Purpose                        |
| ------------ | ------------------------- | ------------------------------ |
| `RELAY_BASE` | `http://10.0.2.2:8080`    | Base URL of the Go relay.      |

Pass overrides with `--dart-define=RELAY_BASE=https://relay.example.com`.

## How it works

- On first launch we mint a device UUID, persist it in the platform
  secure storage, and `POST /auth/register` to obtain a JWT + user id.
- `RelayClient` opens `GET /ws?token=...&last_seq=<maxSeq>` and emits
  decoded `Message`s. Reconnects use exponential backoff capped at 30s,
  and re-read `last_seq` from the local DB before each attempt.
- `ProviderSync` bridges the DAO and the relay:
  - UI writes → `dao.upsert` + provider broadcast.
  - Incoming provider messages → `dao.applyBroadcast`.
  - Incoming connector messages → apply locally, then forward as a
    provider broadcast using this device's identity.

## Layout

```
lib/
  pb/wire.dart                # hand-written protojson types
  core/config.dart            # RELAY_BASE
  core/database/app_db.dart   # drift DB + Records table
  core/database/records_dao.dart
  core/relay/auth.dart        # /auth/register
  core/relay/relay_state.dart # enum
  core/relay/relay_client.dart# WS wrapper w/ backoff
  core/sync/provider_sync.dart
  features/home/home_page.dart
  main.dart
```

## Known limitations in this phase

- Provider role only; connector role is not implemented.
- No offline buffering of outbound messages: if the socket is down the
  local DB is still the source of truth and a later reconnect with
  `last_seq` pulls any missed state, but local-only writes made while
  offline are not re-emitted automatically.
- No tests shipped; `flutter analyze` is the only gate.

[drift]: https://drift.simonbinder.eu/
