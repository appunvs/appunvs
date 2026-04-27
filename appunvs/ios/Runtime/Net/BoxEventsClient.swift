// BoxEventsClient — long-lived SSE consumer for GET /box/events.
//
// Opened once at sign-in (RuntimeApp.SignedInRoot.wireAndLoad) and torn
// down at sign-out.  Receives `bundle_ready` events for any box owned by
// the authenticated user; the consumer typically calls
// `boxStore.refresh()` so Stage's reactive binding to activeBox.bundleURL
// flips and RuntimeView re-mounts the new bundle.
//
// Reconnect: this client keeps trying.  Network drops, server restarts,
// proxy timeouts — all are handled by an internal exponential-backoff
// reconnect loop.  On a fresh (re)connect we yield a `.reconnected`
// element so the caller can `boxStore.refresh()` and pick up any events
// missed while disconnected (publishes during the gap fan out to
// subscribers that don't exist; we recover by polling once).
//
// Heartbeats (`event: heartbeat`) keep the TCP connection alive through
// reverse proxies; they're consumed silently and never reach the caller.
//
// Surface: `events()` returns an AsyncStream the caller drives via a
// `for await` loop inside a `.task { }` modifier.  Cancellation
// propagates: when the consuming Task is cancelled (View disappears,
// .task replaces), AsyncStream's onTermination fires and the internal
// reconnect Task cancels.  No explicit stop() needed — Task lifecycle
// owns it.
import Foundation

struct BoxBundleReadyEvent: Decodable, Sendable {
    let type: String
    let boxID: String
    let version: String
    let uri: String
    let contentHash: String
    let sizeBytes: Int64

    enum CodingKeys: String, CodingKey {
        case type
        case boxID       = "box_id"
        case version
        case uri
        case contentHash = "content_hash"
        case sizeBytes   = "size_bytes"
    }
}

/// What the events() stream yields.
enum BoxStreamEvent: Sendable {
    case bundleReady(BoxBundleReadyEvent)
    /// Emitted on every successful (re)connect AFTER the first.  The
    /// caller uses this as a hint to re-fetch the box list and catch up
    /// on events that may have fired while disconnected.
    case reconnected
}

/// Plain final class — same rationale as AISSEClient: no mutable state
/// the loop needs synchronized access to, and going through an actor
/// would force a hop on every byte read.
final class BoxEventsClient: @unchecked Sendable {
    private let baseURL: URL
    private let session: URLSession
    private let tokenProvider: @Sendable () -> String?

    /// Backoff schedule for reconnect attempts.  Walks the array and
    /// stays at the last value indefinitely.  6 entries totals ~62s
    /// before settling at the 30s cap.
    private let backoff: [TimeInterval] = [1, 2, 5, 10, 20, 30]

    init(
        baseURL: URL = NetConfig.relayBaseURL,
        session: URLSession = .shared,
        tokenProvider: @escaping @Sendable () -> String?
    ) {
        self.baseURL = baseURL
        self.session = session
        self.tokenProvider = tokenProvider
    }

    /// Returns an infinite stream of bundle_ready / reconnected events.
    /// The internal reconnect Task lives until the consumer cancels its
    /// for-await loop (e.g. by Task cancellation on view disappear).
    func events() -> AsyncStream<BoxStreamEvent> {
        AsyncStream { continuation in
            let task = Task { [self] in
                var attempt = 0
                var firstConnect = true
                while !Task.isCancelled {
                    let connected = await runOnce(
                        firstConnect: firstConnect,
                        continuation: continuation,
                    )
                    if Task.isCancelled { break }
                    if connected {
                        attempt = 0
                        firstConnect = false
                    } else {
                        attempt = min(attempt + 1, backoff.count - 1)
                    }
                    let delay = backoff[min(attempt, backoff.count - 1)]
                    try? await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
                }
                continuation.finish()
            }
            continuation.onTermination = { _ in task.cancel() }
        }
    }

    /// One open-stream-and-read attempt.  Returns true if it managed to
    /// connect (so the caller can reset its backoff counter), false if
    /// the request failed before any data arrived.
    private func runOnce(
        firstConnect: Bool,
        continuation: AsyncStream<BoxStreamEvent>.Continuation
    ) async -> Bool {
        var req = URLRequest(url: baseURL.appendingPathComponent("/box/events"))
        req.httpMethod = "GET"
        req.setValue("text/event-stream", forHTTPHeaderField: "Accept")
        if let token = tokenProvider() {
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        do {
            let (bytes, response) = try await session.bytes(for: req)
            guard let http = response as? HTTPURLResponse,
                  (200..<300).contains(http.statusCode) else {
                return false
            }
            // Tell the consumer we're (re)connected — but only after
            // the first connect, so the very first refresh isn't
            // duplicated against the wireAndLoad initial refresh.
            if !firstConnect {
                continuation.yield(.reconnected)
            }

            var event: String = "message"
            for try await line in bytes.lines {
                if Task.isCancelled { break }
                if line.isEmpty {
                    event = "message"
                    continue
                }
                if line.hasPrefix("event: ") {
                    event = String(line.dropFirst(7))
                    continue
                }
                if line.hasPrefix("data: ") {
                    let json = String(line.dropFirst(6))
                    if let value = parse(event: event, json: json) {
                        continuation.yield(value)
                    }
                }
            }
            return true
        } catch {
            return false
        }
    }

    private func parse(event: String, json: String) -> BoxStreamEvent? {
        switch event {
        case "bundle_ready":
            guard let data = json.data(using: .utf8) else { return nil }
            if let decoded = try? JSONDecoder().decode(BoxBundleReadyEvent.self, from: data) {
                return .bundleReady(decoded)
            }
            return nil
        case "heartbeat", "message":
            return nil
        default:
            return nil
        }
    }
}
