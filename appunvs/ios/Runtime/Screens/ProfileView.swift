// ProfileView — account center.  Sections:
//
//   1. Account header (placeholder identity)
//   2. Today's usage (mock numbers)
//   3. Theme override (functional, persisted to UserDefaults)
//   4. Devices (placeholder)
//   5. Footer (sign-out placeholder + version string)
//
// Box list is intentionally NOT here — switching Box lives in the
// Chat tab's BoxSwitcher chip.
import SwiftUI

struct ProfileView: View {
    @EnvironmentObject private var state: AppState
    @EnvironmentObject private var auth: AuthStore

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: Spacing.l) {
                    accountCard
                    usageCard
                    themeCard
                    devicesCard
                    footer
                }
                .padding(Spacing.l)
            }
            .scrollContentBackground(.hidden)
            .background(Theme.bgPage.color)
            .navigationTitle("个人中心")
        }
    }

    // MARK: - Sections

    private var accountCard: some View {
        Card {
            HStack(spacing: Spacing.l) {
                Circle()
                    .fill(Theme.brandPale.color)
                    .frame(width: 56, height: 56)
                    .overlay(
                        Text(initial)
                            .font(.title2.weight(.bold))
                            .foregroundStyle(Theme.brandDark.color)
                    )
                VStack(alignment: .leading, spacing: 2) {
                    Text(displayName)
                        .font(.headline)
                        .foregroundStyle(Theme.textPrimary.color)
                    Text(displayEmail)
                        .font(.subheadline)
                        .foregroundStyle(Theme.textSecondary.color)
                }
                Spacer()
                Badge("Free", tone: .info)
            }
        }
    }

    private var displayEmail: String { auth.me?.email ?? "—" }

    private var displayName: String {
        if let email = auth.me?.email, let at = email.firstIndex(of: "@") {
            return String(email[..<at])
        }
        return "已登录"
    }

    private var initial: String {
        String(displayName.prefix(1)).uppercased()
    }

    private var usageCard: some View {
        Card {
            VStack(alignment: .leading, spacing: Spacing.m) {
                Text("本月用量")
                    .font(.headline)
                    .foregroundStyle(Theme.textPrimary.color)
                QuotaRow(label: "对话", used: 0, cap: 300)
                QuotaRow(label: "存储", used: 0, cap: 5_120, unit: "MB")
            }
        }
    }

    private var themeCard: some View {
        Card {
            VStack(alignment: .leading, spacing: Spacing.s) {
                Text("主题")
                    .font(.headline)
                    .foregroundStyle(Theme.textPrimary.color)
                Picker("", selection: $state.themeOverride) {
                    Text("跟随系统").tag(AppState.ThemeOverride.system)
                    Text("浅色").tag(AppState.ThemeOverride.light)
                    Text("深色").tag(AppState.ThemeOverride.dark)
                }
                .pickerStyle(.segmented)
            }
        }
    }

    private var devicesCard: some View {
        Card(padding: 0) {
            VStack(spacing: 0) {
                Text("设备")
                    .font(.headline)
                    .foregroundStyle(Theme.textPrimary.color)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(Spacing.l)
                Divider().background(Theme.borderDefault.color)
                if let devices = auth.me?.devices, !devices.isEmpty {
                    ForEach(devices) { dev in
                        HStack(spacing: Spacing.m) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(dev.platform)
                                    .font(.body.weight(.semibold))
                                    .foregroundStyle(Theme.textPrimary.color)
                                Text(dev.id)
                                    .font(.caption.monospaced())
                                    .foregroundStyle(Theme.textSecondary.color)
                                    .lineLimit(1)
                                    .truncationMode(.middle)
                            }
                            Spacer()
                        }
                        .padding(Spacing.l)
                        Divider().background(Theme.borderDefault.color)
                    }
                } else {
                    HStack(spacing: Spacing.m) {
                        VStack(alignment: .leading, spacing: 2) {
                            Text("当前设备")
                                .font(.body.weight(.semibold))
                                .foregroundStyle(Theme.textPrimary.color)
                            Text("此刻活跃")
                                .font(.caption)
                                .foregroundStyle(Theme.textSecondary.color)
                        }
                        Spacer()
                        Badge("本机", tone: .info)
                    }
                    .padding(Spacing.l)
                }
            }
        }
    }

    private var footer: some View {
        VStack(spacing: Spacing.s) {
            Button("退出登录") {
                auth.signOut()
            }
            .buttonStyle(.plain)
            .foregroundStyle(Theme.semanticDanger.color)
            Text("appunvs · v0.0.1 (dev)")
                .font(.caption)
                .foregroundStyle(Theme.textSecondary.color)
            #if DEBUG
            // Hidden behind DEBUG so the design-tokens preview ships
            // out of release builds.  Used while iterating on Theme.swift
            // / Typography to see every token rendered in one screen.
            NavigationLink("Design tokens →", destination: TokensPreviewView())
                .font(.caption)
                .foregroundStyle(Theme.textSecondary.color)
            #endif
        }
        .frame(maxWidth: .infinity)
        .padding(.top, Spacing.l)
    }
}

private struct QuotaRow: View {
    let label: String
    let used: Int
    let cap: Int
    var unit: String? = nil

    private var ratio: Double {
        cap == 0 ? 0 : min(1, Double(used) / Double(cap))
    }

    private var fillColor: Color {
        ratio >= 0.9 ? Theme.semanticWarning.color : Theme.brandDark.color
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(label)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(Theme.textPrimary.color)
                Spacer()
                Text("\(used) / \(cap)\(unit.map { " \($0)" } ?? "")")
                    .font(.caption)
                    .foregroundStyle(Theme.textSecondary.color)
            }
            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    Capsule().fill(Theme.bgInput.color)
                    Capsule()
                        .fill(fillColor)
                        .frame(width: geo.size.width * ratio)
                }
            }
            .frame(height: 6)
        }
    }
}

#Preview {
    ProfileView()
        .environmentObject(AppState())
        .environmentObject(AuthStore())
}
