// AISSEClient — Server-Sent Events consumer for POST /ai/turn.
//
// The relay emits one event per frame (`event: token|tool_call|
// tool_result|finished|error`) with a single `data:` JSON line.  We
// parse the byte stream into discrete `AIFrame`s and yield them as an
// `AsyncThrowingStream` so callers can `for try await frame in ...`.
//
// No third-party SSE lib — URLSession's `bytes(for:)` gives us a line
// stream that's good enough for the tiny grammar we emit.
import Foundation

enum AIFrame {
    case token(turnID: String, text: String)
    case toolCall(turnID: String, callID: String, name: String, argsJSON: String)
    case toolResult(turnID: String, callID: String, resultJSON: String, isError: Bool)
    case finished(turnID: String, stopReason: String, tokensIn: Int, tokensOut: Int)
    case error(turnID: String, message: String)
}

struct AITurnRequest: Encodable {
    let boxID: String
    let text: String

    enum CodingKeys: String, CodingKey {
        case boxID = "box_id"
        case text
    }
}

actor AISSEClient {
    private let baseURL: URL
    private let session: URLSession
    private let tokenProvider: @Sendable () -> String?

    init(
        baseURL: URL = NetConfig.relayBaseURL,
        session: URLSession = .shared,
        tokenProvider: @escaping @Sendable () -> String?
    ) {
        self.baseURL = baseURL
        self.session = session
        self.tokenProvider = tokenProvider
    }

    /// Streams frames for a single AI turn.  The stream finishes after a
    /// `.finished` or `.error` frame, or when the underlying connection
    /// closes.
    func turn(boxID: String, text: String) -> AsyncThrowingStream<AIFrame, Error> {
        AsyncThrowingStream { continuation in
            let task = Task {
                do {
                    var req = URLRequest(url: baseURL.appendingPathComponent("/ai/turn"))
                    req.httpMethod = "POST"
                    req.setValue("application/json", forHTTPHeaderField: "Content-Type")
                    req.setValue("text/event-stream", forHTTPHeaderField: "Accept")
                    if let token = tokenProvider() {
                        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
                    }
                    req.httpBody = try JSONEncoder().encode(
                        AITurnRequest(boxID: boxID, text: text)
                    )

                    let (bytes, response) = try await session.bytes(for: req)
                    if let http = response as? HTTPURLResponse,
                       !(200..<300).contains(http.statusCode) {
                        // Drain a small chunk of body for the error message.
                        var buf = Data()
                        for try await b in bytes.prefix(2048) { buf.append(b) }
                        let msg = String(data: buf, encoding: .utf8) ?? "(no body)"
                        throw HTTPError.status(http.statusCode, msg)
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
                            if let frame = parse(event: event, json: json) {
                                continuation.yield(frame)
                                if case .finished = frame { break }
                                if case .error = frame { break }
                            }
                        }
                    }
                    continuation.finish()
                } catch {
                    continuation.finish(throwing: error)
                }
            }
            continuation.onTermination = { _ in task.cancel() }
        }
    }

    // MARK: - parse helpers

    private func parse(event: String, json: String) -> AIFrame? {
        guard let data = json.data(using: .utf8) else { return nil }
        let dict = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any] ?? [:]
        let turnID = dict["turn_id"] as? String ?? ""
        switch event {
        case "token":
            return .token(turnID: turnID, text: dict["text"] as? String ?? "")
        case "tool_call":
            return .toolCall(
                turnID: turnID,
                callID: dict["call_id"] as? String ?? "",
                name: dict["name"] as? String ?? "",
                argsJSON: dict["args_json"] as? String ?? ""
            )
        case "tool_result":
            return .toolResult(
                turnID: turnID,
                callID: dict["call_id"] as? String ?? "",
                resultJSON: dict["result_json"] as? String ?? "",
                isError: dict["is_error"] as? Bool ?? false
            )
        case "finished":
            return .finished(
                turnID: turnID,
                stopReason: dict["stop_reason"] as? String ?? "",
                tokensIn: (dict["tokens_in"] as? Int) ?? 0,
                tokensOut: (dict["tokens_out"] as? Int) ?? 0
            )
        case "error":
            return .error(
                turnID: turnID,
                message: dict["error"] as? String ?? "unknown error"
            )
        default:
            return nil
        }
    }
}
