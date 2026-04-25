// ChatView — placeholder.  The real Chat UI lands in PR C: header chip
// (BoxSwitcher), transcript with Bubble + ToolCall blocks, composer
// pinned to the keyboard.  Today renders an empty state pointing at
// the not-yet-implemented Box-create flow.
import SwiftUI

struct ChatView: View {
    var body: some View {
        ZStack {
            Theme.bgPage.color.ignoresSafeArea()
            VStack(spacing: Spacing.l) {
                Image(systemName: "bubble.left.and.bubble.right")
                    .font(.system(size: 48))
                    .foregroundStyle(Theme.brandDark.color)
                Text("Chat")
                    .font(.title2.weight(.bold))
                    .foregroundStyle(Theme.textPrimary.color)
                Text("待 PR C: 把 Bubble / ToolCall / Composer 从 RN 端口过来。")
                    .font(.callout)
                    .foregroundStyle(Theme.textSecondary.color)
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: 320)
            }
            .padding(Spacing.xxl)
        }
    }
}

#Preview {
    ChatView()
}
