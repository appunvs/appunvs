// ProfileView — minimal account center.  Sections:
//
//   1. Account header (placeholder identity)
//   2. Theme override picker (functional today; persists via AppState)
//
// Usage quotas / devices / billing land in PR C alongside the network
// client.  Box list management is intentionally NOT here — that lives
// in the Chat tab's BoxSwitcher chip.
import SwiftUI

struct ProfileView: View {
    @EnvironmentObject private var state: AppState

    var body: some View {
        NavigationStack {
            List {
                Section {
                    HStack(spacing: Spacing.l) {
                        Circle()
                            .fill(Theme.brandPale.color)
                            .frame(width: 56, height: 56)
                            .overlay(
                                Text("u")
                                    .font(.title2.weight(.bold))
                                    .foregroundStyle(Theme.brandDark.color)
                            )
                        VStack(alignment: .leading, spacing: 2) {
                            Text("未登录用户")
                                .font(.headline)
                                .foregroundStyle(Theme.textPrimary.color)
                            Text("guest@local")
                                .font(.subheadline)
                                .foregroundStyle(Theme.textSecondary.color)
                        }
                        Spacer()
                    }
                    .padding(.vertical, Spacing.xs)
                }

                Section("主题") {
                    Picker("", selection: $state.themeOverride) {
                        Text("跟随系统").tag(AppState.ThemeOverride.system)
                        Text("浅色").tag(AppState.ThemeOverride.light)
                        Text("深色").tag(AppState.ThemeOverride.dark)
                    }
                    .pickerStyle(.segmented)
                }

                Section {
                    Text("appunvs · v0.0.1")
                        .font(.footnote)
                        .foregroundStyle(Theme.textSecondary.color)
                }
            }
            .navigationTitle("个人中心")
            .scrollContentBackground(.hidden)
            .background(Theme.bgPage.color)
        }
    }
}

#Preview {
    ProfileView().environmentObject(AppState())
}
