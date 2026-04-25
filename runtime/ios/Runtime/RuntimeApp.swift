// RuntimeApp — appunvs iOS host shell entry point.
//
// Three top-level tabs (Chat / Stage / Profile).  AppState owns
// host-wide observable state (theme override); MockStore provides
// in-memory boxes + chat fixtures until the network layer lands.
//
// Theme override is held in AppState and applied as
// `.preferredColorScheme()` on the root view; absence of an override
// follows the system setting.
import SwiftUI

@main
struct RuntimeApp: App {
    @StateObject private var appState = AppState()
    @StateObject private var mockStore = MockStore()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(appState)
                .environmentObject(mockStore)
                .preferredColorScheme(appState.preferredColorScheme)
        }
    }
}

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
    RootView()
        .environmentObject(AppState())
        .environmentObject(MockStore())
}
