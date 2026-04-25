// StageView — D2.e ties Stage to the active Box.  When the user
// switches Boxes via the Chat tab's BoxSwitcher, Stage tracks
// `boxStore.activeBox` and reloads RuntimeView with the new box's
// bundle URL.
//
// The bundle URL today is **derived** from the box id (and the
// configured relay base URL) since the box-list endpoint doesn't
// surface BundleRef.uri on the list shape — only the `/box/{id}`
// detail endpoint does.  D3 either fetches that detail when active
// box changes, or `BoxStore` grows a per-box-detail cache.  Either
// way, this view's contract stays "give me the URL for the active
// box and I'll mount it."
import SwiftUI
import RuntimeSDK

struct StageView: View {
    @EnvironmentObject private var boxStore: BoxStore

    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()
            if let url = activeBundleURL {
                RuntimeViewRepresentable(bundleURL: url)
                    .ignoresSafeArea()
            } else {
                noBoxState
            }
        }
    }

    /// Build a bundle URL from the active box.  Today this synthesizes
    /// the relay's `/_artifacts/<box>/<version>/index.bundle` path; D3
    /// fetches the real `BundleRef.uri` from `/box/{id}`.
    private var activeBundleURL: URL? {
        guard let box = boxStore.activeBox else { return nil }
        let version = box.currentVersion.isEmpty ? "draft" : box.currentVersion
        let path = "/_artifacts/\(box.boxID)/\(version)/index.bundle"
        return URL(string: NetConfig.relayBaseURL.absoluteString + path)
    }

    private var noBoxState: some View {
        VStack(spacing: Spacing.l) {
            Image(systemName: "play.rectangle")
                .font(.system(size: 48))
                .foregroundStyle(Color(white: 0.4))
            Text("挑一个 Box")
                .font(.title3.weight(.bold))
                .foregroundStyle(Color.white)
            Text("从 Chat tab 顶上的 Box 切换器选一个，bundle 在这里跑。")
                .font(.callout)
                .foregroundStyle(Color(white: 0.7))
                .multilineTextAlignment(.center)
                .frame(maxWidth: 320)
        }
        .padding(Spacing.xxl)
    }
}

/// SwiftUI bridge for the SDK's UIKit RuntimeView.  Holds a single
/// instance per Stage mount; when `bundleURL` changes (e.g. user
/// switched Boxes), the wrapped view reloads.
private struct RuntimeViewRepresentable: UIViewRepresentable {
    let bundleURL: URL

    func makeUIView(context: Context) -> RuntimeView {
        let view = RuntimeView(frame: .zero)
        view.loadBundle(at: bundleURL, completion: nil)
        return view
    }

    func updateUIView(_ uiView: RuntimeView, context: Context) {
        if uiView.currentBundleURL != bundleURL {
            uiView.loadBundle(at: bundleURL, completion: nil)
        }
    }
}

#Preview {
    StageView()
        .environmentObject(BoxStore(http: HTTPClient(tokenProvider: { nil })))
}
