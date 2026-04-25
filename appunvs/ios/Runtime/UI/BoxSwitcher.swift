// BoxSwitcher — Chat header chip + bottom sheet listing boxes.
// Tapping the chip opens a sheet with the box list (current marked
// with a brand check), and footer actions for "+ New box" and
// "Scan QR" (the latter still disabled until the pairing flow lands).
import SwiftUI

struct BoxSwitcher: View {
    @EnvironmentObject private var store: BoxStore
    @State private var sheetOpen: Bool = false
    @State private var newBoxOpen: Bool = false

    var body: some View {
        Button {
            sheetOpen = true
        } label: {
            HStack(spacing: Spacing.xs) {
                Text(store.activeBox?.title ?? "选择 Box")
                    .font(.body.weight(.semibold))
                    .foregroundStyle(Theme.textPrimary.color)
                    .lineLimit(1)
                Image(systemName: "chevron.down")
                    .font(.caption)
                    .foregroundStyle(Theme.textSecondary.color)
            }
            .padding(.horizontal, Spacing.s)
            .padding(.vertical, Spacing.xs)
            .background(
                RoundedRectangle(cornerRadius: Radius.m)
                    .fill(Color.clear)
            )
        }
        .buttonStyle(.plain)
        .sheet(isPresented: $sheetOpen) {
            BoxSwitcherSheet(
                onCreate: {
                    sheetOpen = false
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
                        newBoxOpen = true
                    }
                },
                onSelect: { box in
                    store.setActive(box)
                    sheetOpen = false
                }
            )
            .presentationDetents([.medium, .large])
            .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $newBoxOpen) {
            NewBoxSheet { title in
                Task { await store.create(title: title) }
                newBoxOpen = false
            }
        }
    }
}

private struct BoxSwitcherSheet: View {
    @EnvironmentObject private var store: BoxStore
    let onCreate: () -> Void
    let onSelect: (BoxWire) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text("我的 Box")
                    .font(.title3.weight(.bold))
                    .foregroundStyle(Theme.textPrimary.color)
                Spacer()
                Button {
                    Task { await store.refresh() }
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .font(.body)
                        .foregroundStyle(Theme.textSecondary.color)
                }
                .buttonStyle(.plain)
            }
            .padding(.horizontal, Spacing.l)
            .padding(.top, Spacing.m)
            .padding(.bottom, Spacing.s)

            ScrollView {
                if store.boxes.isEmpty {
                    Text(store.loading ? "加载中…" : "还没有 Box, 点下方新建一个")
                        .font(.subheadline)
                        .foregroundStyle(Theme.textSecondary.color)
                        .frame(maxWidth: .infinity, alignment: .center)
                        .padding(Spacing.xxl)
                } else {
                    VStack(spacing: 0) {
                        ForEach(store.boxes) { box in
                            BoxRow(
                                box: box,
                                isActive: box.boxID == store.activeBox?.boxID,
                                onTap: { onSelect(box) }
                            )
                            Divider().background(Theme.borderDefault.color)
                        }
                    }
                }
            }

            Button(action: onCreate) {
                HStack(spacing: Spacing.m) {
                    Image(systemName: "plus.circle.fill")
                        .font(.title2)
                        .foregroundStyle(Theme.brandDark.color)
                    Text("新建 Box")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(Theme.textPrimary.color)
                    Spacer()
                }
                .padding(.horizontal, Spacing.l)
                .padding(.vertical, Spacing.m)
            }
            .buttonStyle(.plain)

            HStack(spacing: Spacing.m) {
                Image(systemName: "qrcode.viewfinder")
                    .font(.title2)
                    .foregroundStyle(Theme.textSecondary.color)
                Text("扫码看别人的 app")
                    .font(.body.weight(.semibold))
                    .foregroundStyle(Theme.textSecondary.color)
                Spacer()
                Badge("即将上线", tone: .neutral)
            }
            .padding(.horizontal, Spacing.l)
            .padding(.vertical, Spacing.m)
            .opacity(0.5)
        }
        .background(Theme.bgCard.color)
    }
}

private struct BoxRow: View {
    let box: BoxWire
    let isActive: Bool
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            HStack(spacing: Spacing.m) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(box.title)
                        .font(.body.weight(.semibold))
                        .foregroundStyle(Theme.textPrimary.color)
                        .lineLimit(1)
                    Text("v\(box.currentVersion.isEmpty ? "—" : box.currentVersion)")
                        .font(.caption)
                        .foregroundStyle(Theme.textSecondary.color)
                }
                Spacer()
                Badge(box.state.rawValue, tone: tone(for: box.state))
                if isActive {
                    Image(systemName: "checkmark")
                        .font(.caption.weight(.bold))
                        .foregroundStyle(Theme.brandDark.color)
                }
            }
            .padding(.horizontal, Spacing.l)
            .padding(.vertical, Spacing.m)
            .background(isActive ? Theme.brandPale.color : Color.clear)
        }
        .buttonStyle(.plain)
    }

    private func tone(for state: BoxStateWire) -> BadgeTone {
        switch state {
        case .draft:       return .warning
        case .published:   return .success
        case .archived:    return .neutral
        case .unspecified: return .neutral
        }
    }
}

private struct NewBoxSheet: View {
    let onCreate: (String) -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var title: String = ""

    var body: some View {
        NavigationStack {
            Form {
                Section("名称") {
                    TextField("比如 todo-app", text: $title)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                }
                Section {
                    Text("Box 是一个独立项目, 对话历史和源代码都和它绑定。")
                        .font(.footnote)
                        .foregroundStyle(Theme.textSecondary.color)
                }
            }
            .scrollContentBackground(.hidden)
            .background(Theme.bgPage.color)
            .navigationTitle("新建 Box")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("创建") {
                        let t = title.trimmingCharacters(in: .whitespaces)
                        guard !t.isEmpty else { return }
                        onCreate(t)
                    }
                    .disabled(title.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
        }
        .presentationDetents([.medium])
    }
}
