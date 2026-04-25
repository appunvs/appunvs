// Badge — compact pill for status labels (draft / published / running /
// failed / etc).  Tone is the semantic role; the foreground/background
// pair is derived from theme tokens — never set explicit colors at the
// call site.
import SwiftUI

enum BadgeTone {
    case neutral, info, success, warning, danger
}

struct Badge: View {
    let label: String
    let tone: BadgeTone

    init(_ label: String, tone: BadgeTone = .neutral) {
        self.label = label
        self.tone = tone
    }

    var body: some View {
        Text(label)
            .font(.caption.weight(.semibold))
            .foregroundStyle(foreground)
            .padding(.horizontal, Spacing.s)
            .padding(.vertical, 2)
            .background(
                Capsule().fill(background)
            )
    }

    private var foreground: Color {
        switch tone {
        case .neutral: return Theme.textSecondary.color
        case .info:    return Theme.brandDark.color
        case .success: return Theme.semanticSuccess.color
        case .warning: return Theme.semanticWarning.color
        case .danger:  return Theme.semanticDanger.color
        }
    }

    private var background: Color {
        switch tone {
        case .neutral: return Theme.bgInput.color
        case .info, .success: return Theme.brandPale.color
        case .warning, .danger: return Theme.bgInput.color
        }
    }
}

#Preview {
    HStack(spacing: 8) {
        Badge("draft", tone: .warning)
        Badge("published", tone: .success)
        Badge("running", tone: .info)
        Badge("failed", tone: .danger)
        Badge("archived", tone: .neutral)
    }
    .padding()
}
