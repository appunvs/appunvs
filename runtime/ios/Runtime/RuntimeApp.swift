// RuntimeApp — appunvs iOS host shell entry point.
//
// The host has three tabs (Chat / Stage / Profile) — the same information
// architecture as the prior Expo prototype, but rendered with native
// SwiftUI primitives. The Stage tab is currently a placeholder; a future
// SubRuntime native module will mount a sandboxed Hermes runtime view
// here to render AI-authored bundles.
//
// Theme override is held in `AppState` and applied as
// `.preferredColorScheme()` on the root view; absence of an override
// follows the system setting via `@Environment(\.colorScheme)`.
import SwiftUI

@main
struct RuntimeApp: App {
    @StateObject private var state = AppState()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(state)
                .preferredColorScheme(state.preferredColorScheme)
        }
    }
}

/// The three top-level tabs.  Tab order and labels are intentionally
/// stable across breakpoints and devices — landscape iPad picks up the
/// same tab bar.
struct RootView: View {
    @State private var selection: Tab = .chat

    enum Tab: Hashable { case chat, stage, profile }

    var body: some View {
        TabView(selection: $selection) {
            ChatView()
                .tabItem {
                    Label("Chat", systemImage: "bubble.left.and.bubble.right")
                }
                .tag(Tab.chat)

            StageView()
                .tabItem {
                    Label("Stage", systemImage: "play.rectangle")
                }
                .tag(Tab.stage)

            ProfileView()
                .tabItem {
                    Label("Profile", systemImage: "person.crop.circle")
                }
                .tag(Tab.profile)
        }
        .tint(Theme.brandDark.color)
    }
}

#Preview {
    RootView().environmentObject(AppState())
}
