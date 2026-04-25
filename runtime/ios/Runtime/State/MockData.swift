// MockData — in-memory fixtures used while the network layer is still
// being designed.  Real /box and /pair clients arrive in a follow-up
// PR; until then, the UI renders against these so screens behave end-
// to-end without a live relay.
//
// Replace consumers with the real network store at that point — these
// types intentionally mirror the proto shapes (`box.proto`, `pair.proto`)
// so the swap is field-for-field.
import Foundation

enum BoxState: String {
    case draft, published, archived
}

struct Box: Identifiable, Hashable {
    let id: String
    let title: String
    let state: BoxState
    let currentVersion: String
    let updatedAt: Date
}

struct ChatMessage: Identifiable, Hashable {
    enum Role { case user, assistant, system }
    let id: UUID
    let role: Role
    let text: String
    let pending: Bool
}

@MainActor
final class MockStore: ObservableObject {
    @Published var boxes: [Box]
    @Published var activeBox: Box?
    @Published var messages: [ChatMessage]

    init() {
        let demo = Box(
            id: "box_demo",
            title: "demo-counter",
            state: .draft,
            currentVersion: "",
            updatedAt: Date()
        )
        self.boxes = [
            demo,
            Box(id: "box_todo",   title: "todo-app",     state: .published, currentVersion: "v3", updatedAt: Date().addingTimeInterval(-3600)),
            Box(id: "box_color",  title: "color-picker", state: .archived,  currentVersion: "v1", updatedAt: Date().addingTimeInterval(-86400)),
        ]
        self.activeBox = demo
        self.messages = [
            ChatMessage(id: UUID(), role: .user, text: "做一个计数器 app", pending: false),
            ChatMessage(id: UUID(), role: .assistant, text: "好的,我来写。先建一个 index.tsx 加 Button。", pending: false),
        ]
    }

    func setActive(_ box: Box) {
        activeBox = box
    }

    func appendUser(_ text: String) {
        messages.append(ChatMessage(id: UUID(), role: .user, text: text, pending: false))
        messages.append(ChatMessage(id: UUID(), role: .assistant, text: "（模拟回复 — 真 AI 在 PR D 接入）", pending: false))
    }

    func createBox(title: String) -> Box {
        let b = Box(
            id: "box_\(UUID().uuidString.prefix(8).lowercased())",
            title: title,
            state: .draft,
            currentVersion: "",
            updatedAt: Date()
        )
        boxes.insert(b, at: 0)
        activeBox = b
        return b
    }
}
