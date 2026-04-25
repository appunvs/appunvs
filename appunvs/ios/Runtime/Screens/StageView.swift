// StageView — D2.b smoke test.  The host now links RuntimeSDK
// (built from runtime/sdk/ios/) and calls into its hello function to
// prove the linkage works end-to-end.
//
// D2.c widens the SDK to expose a real RuntimeView; D2.d mounts that
// here in place of the hello text.  D2.e wires the active Box's
// bundle URL through to RuntimeView.loadBundle(...).
import SwiftUI
import RuntimeSDK

struct StageView: View {
    private var sdkGreeting: String {
        // runtime_sdk_hello() returns a static C string — no free needed.
        guard let cstr = runtime_sdk_hello() else { return "(null)" }
        return String(cString: cstr)
    }

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
                Text(sdkGreeting)
                    .font(.callout.monospaced())
                    .foregroundStyle(Color(white: 0.7))
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: 320)
                Text("D2.c will replace this with a real RuntimeView mount.")
                    .font(.caption)
                    .foregroundStyle(Color(white: 0.5))
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
