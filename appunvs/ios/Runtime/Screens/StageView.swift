// StageView — D2.d wires the host Stage tab to mount the runtime
// SDK's RuntimeView (a UIView subclass) via UIViewRepresentable.
//
// Today the bundle URL is hardcoded so the host can be visually
// verified end-to-end without the relay / Box flow.  D2.e replaces
// the hardcoded URL with `boxStore.activeBox?.bundleURL` and reacts
// to changes (`.onChange(of: ...)` → reset + reload).
import SwiftUI
import RuntimeSDK

struct StageView: View {
    /// Hardcoded for D2.d.  D2.e binds this to BoxStore.activeBox's
    /// bundle URL and reacts to changes.
    private let demoBundleURL = URL(string: "https://relay.example/_artifacts/box_demo/v1/index.bundle")!

    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()
            RuntimeViewRepresentable(bundleURL: demoBundleURL)
                .ignoresSafeArea()
        }
    }
}

/// SwiftUI bridge for the SDK's UIKit RuntimeView.  Holds a single
/// instance per Stage mount; updates to `bundleURL` cause the wrapped
/// view to reload.
private struct RuntimeViewRepresentable: UIViewRepresentable {
    let bundleURL: URL

    func makeUIView(context: Context) -> RuntimeView {
        let view = RuntimeView(frame: .zero)
        // ObjC method `loadBundleAtURL:completion:` imports as
        // `loadBundle(at:completion:)` (Swift drops the "URL"
        // suffix since it's redundant with the parameter type).
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
}
