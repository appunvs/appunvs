// EmptyState — canonical zero-state shape.  Title + optional hint +
// optional action button slot.  Use this on every screen that has a
// "nothing yet" path (no boxes, no bundle, no chat history).
import SwiftUI

struct EmptyState<Action: View>: View {
    let title: String
    let hint: String?
    let action: () -> Action

    init(
        title: String,
        hint: String? = nil,
        @ViewBuilder action: @escaping () -> Action = { EmptyView() }
    ) {
        self.title = title
        self.hint = hint
        self.action = action
    }

    var body: some View {
        VStack(spacing: Spacing.m) {
            Text(title)
                .font(.title2.weight(.bold))
                .foregroundStyle(Theme.textPrimary.color)
                .multilineTextAlignment(.center)
            if let hint {
                Text(hint)
                    .font(.callout)
                    .foregroundStyle(Theme.textSecondary.color)
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: 360)
            }
            action()
                .padding(.top, Spacing.m)
        }
        .padding(Spacing.xxl)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

#Preview {
    EmptyState(
        title: "选个 Box 开始",
        hint: "每个 Box 是一个独立项目，对话历史与代码都和它绑定。",
        action: {
            Button("新建 Box") {}
                .buttonStyle(.borderedProminent)
        }
    )
}
