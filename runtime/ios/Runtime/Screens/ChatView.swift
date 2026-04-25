// ChatView — Chat tab.  Header carries the BoxSwitcher chip; the body
// is a scrolling transcript of Bubbles; the footer is the composer
// pinned over the keyboard.
//
// AI is mock today (the assistant reply is a fixed string from
// MockStore.appendUser).  Real /ai/turn SSE consumption arrives in
// a follow-up PR alongside the network client.
import SwiftUI

struct ChatView: View {
    @EnvironmentObject private var store: MockStore
    @State private var draft: String = ""

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
        if store.activeBox == nil {
            EmptyState(
                title: "选个 Box 开始",
                hint: "每个 Box 是一个独立项目，对话历史与代码都和它绑定。",
                action: { EmptyView() }
            )
        } else if store.messages.isEmpty {
            EmptyState(
                title: "和 AI 说点什么",
                hint: "比如\"做一个计数器 app\"。",
                action: { EmptyView() }
            )
        } else {
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: Spacing.s) {
                        ForEach(store.messages) { msg in
                            Bubble(role: msg.role.bubbleRole, text: msg.text, pending: msg.pending)
                                .id(msg.id)
                        }
                    }
                    .padding(Spacing.l)
                }
                // 1-arg closure: iOS 16 supports only the (newValue) form.
                // The 2-arg (oldValue, newValue) variant is iOS 17+.
                .onChange(of: store.messages.count) { _ in
                    if let last = store.messages.last {
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
                Text("发送")
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
            .disabled(trimmed.isEmpty)
            .opacity(trimmed.isEmpty ? 0.5 : 1)
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

    private func send() {
        guard !trimmed.isEmpty else { return }
        store.appendUser(trimmed)
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

#Preview {
    ChatView()
        .environmentObject(MockStore())
}
