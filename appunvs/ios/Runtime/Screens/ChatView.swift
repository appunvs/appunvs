// ChatView — Chat tab.  Header carries the BoxSwitcher chip; the body
// is a scrolling transcript of Bubbles; the footer is the composer
// pinned over the keyboard.
//
// Backed by ChatStore (real /ai/turn SSE) and BoxStore (real /box).
import SwiftUI

struct ChatView: View {
    @EnvironmentObject private var boxStore: BoxStore
    @EnvironmentObject private var chatStore: ChatStore
    @State private var draft: String = ""

    private var messages: [ChatMessage] {
        chatStore.messages(for: boxStore.activeBox?.boxID)
    }

    var body: some View {
        VStack(spacing: 0) {
            header
            Divider().background(Theme.borderDefault.color)
            transcript
            composer
        }
        .background(Theme.bgPage.color)
    }

    // MARK: - Sections

    private var header: some View {
        HStack {
            BoxSwitcher()
            Spacer()
        }
        .padding(.horizontal, Spacing.l)
        .padding(.vertical, Spacing.s)
        .background(Theme.bgPage.color)
    }

    @ViewBuilder
    private var transcript: some View {
        if boxStore.activeBox == nil {
            EmptyState(
                title: "选个 Box 开始",
                hint: "每个 Box 是一个独立项目，对话历史与代码都和它绑定。",
                action: { EmptyView() }
            )
        } else if messages.isEmpty {
            EmptyState(
                title: "和 AI 说点什么",
                hint: "比如\"做一个计数器 app\"。",
                action: { EmptyView() }
            )
        } else {
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: Spacing.s) {
                        ForEach(messages) { msg in
                            Bubble(role: msg.role.bubbleRole, text: msg.text, pending: msg.pending)
                                .id(msg.id)
                        }
                    }
                    .padding(Spacing.l)
                }
                // 1-arg closure: iOS 16 supports only the (newValue) form.
                // The 2-arg (oldValue, newValue) variant is iOS 17+.
                .onChange(of: messages.count) { _ in
                    if let last = messages.last {
                        withAnimation { proxy.scrollTo(last.id, anchor: .bottom) }
                    }
                }
            }
        }
    }

    private var composer: some View {
        HStack(alignment: .bottom, spacing: Spacing.s) {
            TextField("描述一个改动…", text: $draft, axis: .vertical)
                .lineLimit(1...4)
                .padding(.horizontal, Spacing.m)
                .padding(.vertical, Spacing.s)
                .background(
                    RoundedRectangle(cornerRadius: Radius.m)
                        .fill(Theme.bgInput.color)
                )
                .overlay(
                    RoundedRectangle(cornerRadius: Radius.m)
                        .stroke(Theme.borderDefault.color, lineWidth: 1)
                )
            Button(action: send) {
                Text(chatStore.sending ? "…" : "发送")
                    .font(.body.weight(.semibold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, Spacing.l)
                    .padding(.vertical, Spacing.s)
                    .background(
                        RoundedRectangle(cornerRadius: Radius.m)
                            .fill(Theme.brandDark.color)
                    )
            }
            .buttonStyle(.plain)
            .disabled(!canSend)
            .opacity(canSend ? 1 : 0.5)
        }
        .padding(Spacing.s)
        .background(
            Theme.bgCard.color
                .overlay(
                    Rectangle()
                        .frame(height: 1)
                        .foregroundStyle(Theme.borderDefault.color),
                    alignment: .top
                )
        )
    }

    private var trimmed: String {
        draft.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var canSend: Bool {
        !trimmed.isEmpty && boxStore.activeBox != nil && !chatStore.sending
    }

    private func send() {
        guard let box = boxStore.activeBox, !trimmed.isEmpty else { return }
        chatStore.send(boxID: box.boxID, text: trimmed)
        draft = ""
    }
}

private extension ChatMessage.Role {
    var bubbleRole: BubbleRole {
        switch self {
        case .user:      return .user
        case .assistant: return .assistant
        case .system:    return .system
        }
    }
}
