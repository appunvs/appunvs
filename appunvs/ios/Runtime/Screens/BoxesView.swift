// BoxesView — entry tab.  Shows the user's box list and lets them
// drill into Chat for a specific box.  Replaces the previous Chat tab
// (which mixed box selection with conversation in a single screen).
//
// Structure mirrors the design handoff (docs/competitive-landscape.md
// reference: appunvs-prototype/src/app-ios.jsx → BoxListScreenIOS):
//   - 32pt bold title "我的 Box" + "+" create button
//   - vertically stacked box cards: 40x40 brand-pale icon + title +
//     mono version + state badge + chevron
//   - faded "扫码看别人的 app · 即将上线" hint at the bottom
//
// Selecting a row pushes a ChatView onto the local NavigationStack; the
// back chevron returns here.  Stage tab still observes
// boxStore.activeBox so the user's last-tapped box is what's running.
import SwiftUI

struct BoxesView: View {
    @EnvironmentObject private var boxStore: BoxStore

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: Spacing.l) {
                    headerRow
                    boxList
                    scanHint
                }
                .padding(Spacing.l)
            }
            .background(Theme.bgPage.color)
            .navigationBarHidden(true)
            .navigationDestination(for: BoxWire.self) { box in
                ChatView(box: box)
            }
        }
    }

    // MARK: - Sections

    private var headerRow: some View {
        HStack(alignment: .firstTextBaseline) {
            Text("我的 Box")
                .appFont(Typography.display)
                .foregroundStyle(Theme.textPrimary.color)
            Spacer()
            Button(action: createBox) {
                Image(systemName: "plus")
                    .font(.system(size: 22, weight: .light))
                    .foregroundStyle(Theme.brandDark.color)
                    .frame(width: 32, height: 32)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("新建 Box")
        }
    }

    @ViewBuilder
    private var boxList: some View {
        if boxStore.boxes.isEmpty {
            EmptyState(
                title: "没有 Box",
                hint: "点右上角 + 新建一个项目。",
                action: { EmptyView() }
            )
        } else {
            VStack(spacing: Spacing.s) {
                ForEach(boxStore.boxes) { box in
                    NavigationLink(value: box) {
                        BoxRow(box: box, isActive: box.boxID == boxStore.activeBox?.boxID)
                    }
                    .buttonStyle(.plain)
                    .simultaneousGesture(TapGesture().onEnded {
                        boxStore.setActive(box)
                    })
                }
            }
        }
    }

    private var scanHint: some View {
        HStack(spacing: Spacing.m) {
            Image(systemName: "qrcode.viewfinder")
                .font(.system(size: 18))
                .foregroundStyle(Theme.textSecondary.color)
            Text("扫码看别人的 app")
                .appFont(Typography.body)
                .foregroundStyle(Theme.textSecondary.color)
            Spacer()
            Badge("即将上线", tone: .neutral)
        }
        .opacity(0.55)
        .padding(.top, Spacing.s)
    }

    // MARK: - Actions

    private func createBox() {
        Task { await boxStore.create(title: "new-box") }
    }
}

/// One row in the box list — 40×40 icon, title + version, state badge,
/// chevron.  Tappable wrapper provided by the parent's NavigationLink.
private struct BoxRow: View {
    let box: BoxWire
    let isActive: Bool

    var body: some View {
        HStack(spacing: Spacing.m) {
            iconBadge
            VStack(alignment: .leading, spacing: 2) {
                Text(box.title)
                    .appFont(Typography.bodyEmphasis)
                    .foregroundStyle(Theme.textPrimary.color)
                Text("v\(box.currentVersion.isEmpty ? "—" : box.currentVersion)")
                    .appFont(Typography.caption)
                    .foregroundStyle(Theme.textSecondary.color)
                    .monospaced()
            }
            Spacer()
            stateBadge
            Image(systemName: "chevron.right")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(Theme.textSecondary.color)
        }
        .padding(.horizontal, Spacing.l)
        .padding(.vertical, Spacing.m)
        .background(
            RoundedRectangle(cornerRadius: Radius.l)
                .fill(Theme.bgCard.color)
        )
        .overlay(
            RoundedRectangle(cornerRadius: Radius.l)
                .stroke(
                    isActive ? Theme.brandDark.color : Theme.borderDefault.color,
                    lineWidth: isActive ? 1 : 0.5
                )
        )
    }

    private var iconBadge: some View {
        RoundedRectangle(cornerRadius: Radius.m)
            .fill(Theme.brandPale.color)
            .frame(width: 40, height: 40)
            .overlay(
                Image(systemName: "shippingbox")
                    .font(.system(size: 18, weight: .regular))
                    .foregroundStyle(Theme.brandDark.color)
            )
    }

    private var stateBadge: some View {
        Badge(
            box.state.rawValue,
            tone: badgeTone(for: box.state)
        )
    }

    private func badgeTone(for state: BoxStateWire) -> BadgeTone {
        switch state {
        case .published: return .success
        case .draft:     return .warning
        case .archived:  return .neutral
        case .unspecified: return .neutral
        }
    }
}

#Preview {
    BoxesView()
        .environmentObject(BoxStore(http: HTTPClient(tokenProvider: { nil })))
}
