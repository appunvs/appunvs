// ChatView — chat detail screen reached by drilling into a box from
// BoxesView (the new entry tab).  Used to be the root Chat tab; now
// it's pushed onto BoxesView's NavigationStack.
//
// Header: native NavigationStack back chevron + box title.  Body: the
// scrolling transcript (Bubble rows).  Footer: composer pinned at the
// bottom.  Same store wiring as before — ChatStore.send(boxID:text:)
// drives the SSE stream; BoxStore.activeBox tracks the currently
// selected box so Stage knows what to mount.
import SwiftUI

struct ChatView: View {
    let box: BoxWire

    @EnvironmentObject private var boxStore: BoxStore
    @EnvironmentObject private var chatStore: ChatStore
    @State private var draft: String = ""

    private var messages: [ChatMessage] {
        chatStore.messages(for: box.boxID)
    }

    var body: some View {
        VStack(spacing: 0) {
            transcript
            composer
        }
        .background(Theme.bgPage.color)
        .navigationTitle(box.title)
        .navigationBarTitleDisplayMode(.inline)
        .onAppear { boxStore.setActive(box) }
    }

    @ViewBuilder
    private var transcript: some View {
        if messages.isEmpty {
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
                    .appFont(Typography.bodyEmphasis)
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
        !trimmed.isEmpty && !chatStore.sending
    }

    private func send() {
        guard !trimmed.isEmpty else { return }
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
