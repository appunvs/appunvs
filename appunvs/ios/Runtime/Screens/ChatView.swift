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

    @EnvironmentObject private var appState: AppState
    @EnvironmentObject private var boxStore: BoxStore
    @EnvironmentObject private var chatStore: ChatStore
    @State private var draft: String = ""
    /// Per-turn model override.  `nil` means "use AppState.defaultModelID"
    /// (the user's Profile pick); the chat-header chip writes here.
    /// Reset to nil on box change so each box's session starts from the
    /// global default.
    @State private var overrideModelID: String? = nil

    private var messages: [ChatMessage] {
        chatStore.messages(for: box.boxID)
    }

    /// Resolved model id used by the next send().  Per-turn override wins
    /// over the AppState default.
    private var activeModelID: String {
        overrideModelID ?? appState.defaultModelID
    }

    private var activeModel: ChatModel {
        ModelCatalog.find(id: activeModelID) ?? ModelCatalog.fallback
    }

    var body: some View {
        VStack(spacing: 0) {
            transcript
            composer
        }
        .background(Theme.bgPage.color)
        .navigationTitle(box.title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                modelChip
            }
        }
        .onAppear { boxStore.setActive(box) }
        .onChange(of: box.boxID) { _ in overrideModelID = nil }
    }

    /// Trailing-toolbar Menu showing the active model.  Picker entries
    /// switch overrideModelID; the chip falls back to AppState.defaultModelID
    /// when no override is set.  v0 catalog has one item, but the wiring
    /// is full so adding entries to ModelCatalog surfaces them here.
    private var modelChip: some View {
        Menu {
            ForEach(ModelCatalog.all) { m in
                Button {
                    overrideModelID = m.id
                } label: {
                    if m.id == activeModelID {
                        Label(m.name, systemImage: "checkmark")
                    } else {
                        Text(m.name)
                    }
                }
            }
            if overrideModelID != nil {
                Divider()
                Button("跟随默认") { overrideModelID = nil }
            }
        } label: {
            HStack(spacing: 4) {
                Text(activeModel.name)
                    .font(.caption.weight(.semibold))
                Image(systemName: "chevron.down")
                    .font(.caption2)
            }
            .padding(.horizontal, Spacing.s)
            .padding(.vertical, 4)
            .background(
                Capsule().fill(Theme.bgInput.color)
            )
            .overlay(
                Capsule().stroke(Theme.borderDefault.color, lineWidth: 1)
            )
            .foregroundStyle(Theme.textPrimary.color)
        }
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
        chatStore.send(boxID: box.boxID, text: trimmed, model: activeModelID)
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
