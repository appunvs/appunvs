// ChatStore — chat transcript + AI streaming.
//
// Per active box we keep a separate transcript in memory (no relay-
// side persistence yet — `ai_turns` is server-internal for replay /
// audit).  When the user sends a message:
//
//   1. append a user bubble immediately
//   2. append a pending assistant bubble
//   3. open /ai/turn SSE; on each `token` frame extend the bubble's
//      text; flip pending=false on `finished`
//
// Tool-call frames are surfaced as system bubbles ("$tool: …") so the
// user sees what the agent is doing; we'll switch to a richer in-line
// representation once Stage lands.
import Foundation
import SwiftUI

struct ChatMessage: Identifiable, Hashable {
    enum Role { case user, assistant, system }
    let id: UUID
    let role: Role
    var text: String
    var pending: Bool

    init(id: UUID = UUID(), role: Role, text: String, pending: Bool = false) {
        self.id = id
        self.role = role
        self.text = text
        self.pending = pending
    }
}

@MainActor
final class ChatStore: ObservableObject {
    @Published private(set) var messagesByBox: [String: [ChatMessage]] = [:]
    @Published private(set) var sending: Bool = false
    @Published var lastError: String?

    private var sse: AISSEClient
    private var currentTask: Task<Void, Never>?

    init(sse: AISSEClient) {
        self.sse = sse
    }

    /// Swap the underlying SSE client (see BoxStore.rebind for context).
    func rebind(sse: AISSEClient) {
        self.sse = sse
    }

    func messages(for boxID: String?) -> [ChatMessage] {
        guard let id = boxID else { return [] }
        return messagesByBox[id] ?? []
    }

    func send(boxID: String, text: String) {
        currentTask?.cancel()
        sending = true
        var transcript = messagesByBox[boxID] ?? []
        let user = ChatMessage(role: .user, text: text)
        var assistant = ChatMessage(role: .assistant, text: "", pending: true)
        transcript.append(user)
        transcript.append(assistant)
        messagesByBox[boxID] = transcript

        let assistantID = assistant.id
        currentTask = Task {
            do {
                for try await frame in await sse.turn(boxID: boxID, text: text) {
                    switch frame {
                    case .token(_, let text):
                        appendToken(boxID: boxID, id: assistantID, text: text)
                    case .toolCall(_, _, let name, let argsJSON):
                        appendSystem(boxID: boxID, text: "› \(name) \(argsJSON)")
                    case .toolResult(_, _, _, let isError):
                        if isError {
                            appendSystem(boxID: boxID, text: "✗ tool failed")
                        }
                    case .finished:
                        finalize(boxID: boxID, id: assistantID)
                    case .error(_, let message):
                        finalize(boxID: boxID, id: assistantID)
                        appendSystem(boxID: boxID, text: "× \(message)")
                        lastError = message
                    }
                }
                finalize(boxID: boxID, id: assistantID)
            } catch {
                finalize(boxID: boxID, id: assistantID)
                lastError = error.localizedDescription
                appendSystem(boxID: boxID, text: "× \(error.localizedDescription)")
            }
            sending = false
        }
        _ = assistant
    }

    func cancel() {
        currentTask?.cancel()
        sending = false
    }

    func clear(boxID: String) {
        messagesByBox[boxID] = []
    }

    // MARK: - private

    private func appendToken(boxID: String, id: UUID, text: String) {
        var transcript = messagesByBox[boxID] ?? []
        guard let idx = transcript.firstIndex(where: { $0.id == id }) else { return }
        transcript[idx].text += text
        messagesByBox[boxID] = transcript
    }

    private func appendSystem(boxID: String, text: String) {
        var transcript = messagesByBox[boxID] ?? []
        transcript.append(ChatMessage(role: .system, text: text))
        messagesByBox[boxID] = transcript
    }

    private func finalize(boxID: String, id: UUID) {
        var transcript = messagesByBox[boxID] ?? []
        guard let idx = transcript.firstIndex(where: { $0.id == id }) else { return }
        transcript[idx].pending = false
        if transcript[idx].text.isEmpty {
            transcript[idx].text = "(no response)"
        }
        messagesByBox[boxID] = transcript
    }
}
