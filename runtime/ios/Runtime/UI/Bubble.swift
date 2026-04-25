// Bubble — chat message.  User messages right-align with brand fill;
// assistant / system messages left-align with bordered surface.  Soft
// corner radius for both; the per-corner "tail" trim used in the prior
// RN port wants `UnevenRoundedRectangle` which is iOS 16.4 only — we
// target iOS 16.0, so a uniform RoundedRectangle is the compromise.
import SwiftUI

enum BubbleRole {
    case user, assistant, system
}

struct Bubble: View {
    let role: BubbleRole
    let text: String
    let pending: Bool

    init(role: BubbleRole, text: String, pending: Bool = false) {
        self.role = role
        self.text = text
        self.pending = pending
    }

    var body: some View {
        HStack {
            if role == .user { Spacer(minLength: 0) }
            Text(displayText)
                .font(.body)
                .foregroundStyle(foreground)
                .padding(.horizontal, Spacing.l)
                .padding(.vertical, Spacing.m)
                .background(
                    RoundedRectangle(cornerRadius: Radius.xl, style: .continuous)
                        .fill(background)
                )
                .overlay(
                    role == .user
                        ? nil
                        : RoundedRectangle(cornerRadius: Radius.xl, style: .continuous)
                            .stroke(Theme.borderDefault.color, lineWidth: 1)
                )
                .frame(maxWidth: .infinity, alignment: role == .user ? .trailing : .leading)
            if role != .user { Spacer(minLength: 0) }
        }
    }

    private var displayText: String {
        if text.isEmpty && pending { return "…" }
        return text
    }

    private var foreground: Color {
        if role == .user {
            return Color.white
        }
        return Theme.textPrimary.color
    }

    private var background: Color {
        switch role {
        case .user:      return Theme.brandDark.color
        case .assistant: return Theme.bgCard.color
        case .system:    return Theme.bgInput.color
        }
    }
}

#Preview {
    VStack(alignment: .leading, spacing: 8) {
        Bubble(role: .user, text: "做一个计数器 app")
        Bubble(role: .assistant, text: "好的,我来写。")
        Bubble(role: .assistant, text: "", pending: true)
    }
    .padding()
    .background(Theme.bgPage.color)
}
