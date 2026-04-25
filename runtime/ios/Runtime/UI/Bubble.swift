// Bubble — chat message.  User messages right-align with brand fill;
// assistant / system messages left-align with bordered surface.  Max
// width caps at 85% of the parent so two long bubbles can sit side by
// side on the iPad split view.
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
                    UnevenRoundedRectangle(
                        cornerRadii: corners,
                        style: .continuous
                    )
                    .fill(background)
                )
                .overlay(
                    role == .user
                        ? nil
                        : UnevenRoundedRectangle(
                            cornerRadii: corners,
                            style: .continuous
                        )
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

    /// User bubbles get a tighter top-right corner (the "tail" side);
    /// assistant bubbles get a tighter top-left.  The other three
    /// corners stay at radius.xl for the soft "pillow" feel.
    private var corners: RectangleCornerRadii {
        if role == .user {
            return .init(topLeading: Radius.xl, bottomLeading: Radius.xl,
                         bottomTrailing: Radius.xl, topTrailing: Radius.s)
        } else {
            return .init(topLeading: Radius.s, bottomLeading: Radius.xl,
                         bottomTrailing: Radius.xl, topTrailing: Radius.xl)
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
