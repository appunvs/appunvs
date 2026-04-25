// StageView — placeholder.  Once the SubRuntime native module lands
// (PR D), this view hosts a `SubRuntimeView` that mounts a sandboxed
// Hermes runtime to render the active Box's bundle.  Empty state
// stays similar to the Expo prototype: "no bundle loaded yet".
import SwiftUI

struct StageView: View {
    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()
            VStack(spacing: Spacing.l) {
                Image(systemName: "play.rectangle.fill")
                    .font(.system(size: 48))
                    .foregroundStyle(Theme.brandLight.color)
                Text("Stage")
                    .font(.title2.weight(.bold))
                    .foregroundStyle(Color.white)
                Text("待 PR D: SubRuntime native module 接入。")
                    .font(.callout)
                    .foregroundStyle(Color(white: 0.7))
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: 320)
            }
            .padding(Spacing.xxl)
        }
    }
}

#Preview {
    StageView()
}
