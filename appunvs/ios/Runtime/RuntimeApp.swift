// RuntimeApp — appunvs iOS host shell entry point.
//
// AppState owns host-wide observable state (theme override).  AuthStore
// owns the device-token lifecycle (Keychain-backed).  BoxStore + ChatStore
// are constructed *inside* the auth-gated branch so they share the
// AuthStore's HTTPClient / token rotation closure.
//
// Theme override is held in AppState and applied as
// `.preferredColorScheme()` on the root view; absence of an override
// follows the system setting.
import SwiftUI

@main
struct RuntimeApp: App {
    @StateObject private var appState = AppState()
    @StateObject private var auth = AuthStore()

    var body: some Scene {
        WindowGroup {
            Gate()
                .environmentObject(appState)
                .environmentObject(auth)
                .preferredColorScheme(appState.preferredColorScheme)
                .task { await auth.boot() }
        }
    }
}

/// Auth gate — bootstraps once, then either presents `LoginView` or the
/// signed-in `RootView` (which constructs the real stores).
private struct Gate: View {
    @EnvironmentObject private var auth: AuthStore

    var body: some View {
        switch auth.phase {
        case .bootstrapping:
            ZStack {
                Theme.bgPage.color.ignoresSafeArea()
                ProgressView().tint(Theme.brandDark.color)
            }
        case .signedOut:
            LoginView()
        case .signedIn:
            SignedInRoot()
        }
    }
}

private struct SignedInRoot: View {
    @EnvironmentObject private var auth: AuthStore
    @StateObject private var boxStore: BoxStore
    @StateObject private var chatStore: ChatStore

    init() {
        // We need access to the EnvironmentObject auth at init time to
        // share its HTTPClient.  Workaround: the parent (`Gate`) only
        // constructs SignedInRoot once `auth.phase == .signedIn`, but
        // SwiftUI's @StateObject init pattern can't read environment
        // values directly.  We resolve via a transient instance in
        // `onAppear` — see `wireStores` below.
        //
        // For the StateObject defaults we wire dummies that get replaced
        // immediately on first appear.
        let placeholder = HTTPClient(tokenProvider: { nil })
        _boxStore = StateObject(wrappedValue: BoxStore(http: placeholder))
        _chatStore = StateObject(
            wrappedValue: ChatStore(sse: AISSEClient(tokenProvider: { nil }))
        )
    }

    var body: some View {
        RootView()
            .environmentObject(boxStore)
            .environmentObject(chatStore)
            .task { await wireAndLoad() }
    }

    /// Replaces the placeholder stores' clients with ones backed by the
    /// real AuthStore, then triggers an initial /box list refresh.
    private func wireAndLoad() async {
        // Construct real stores against the live auth client.  We can't
        // assign to @StateObject after init, so the running BoxStore /
        // ChatStore use the auth.sharedHTTP via this in-place reset:
        // BoxStore re-issues its api once, ChatStore re-issues its sse.
        //
        // Note: BoxStore / ChatStore expose `api`/`sse` as private; we
        // route through dedicated `rebind` methods below to keep the
        // surface narrow.  Simpler than juggling factories at top level.
        boxStore.rebind(http: auth.sharedHTTP)
        chatStore.rebind(sse: auth.aiClient())
        await boxStore.refresh()
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
    LoginView()
        .environmentObject(AppState())
        .environmentObject(AuthStore())
}
