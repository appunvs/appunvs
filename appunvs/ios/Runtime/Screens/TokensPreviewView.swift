// TokensPreviewView — reachable from Profile in DEBUG builds, renders
// every design token in one scroll surface so iterating on the design
// system is a tight loop: change a value in Theme.swift, hit ⌘B, see
// the new value here.
//
// Sections:
//   - Color tokens: brand / text / surface / semantic, each as a swatch
//     row showing token name + hex pair (light, dark) + visible color
//   - Spacing scale: a stack of bars whose width corresponds to the
//     scale value
//   - Radius scale: rounded squares at each radius
//   - Typography scale: a sample sentence in each token
//
// Not in scope here (defer to the design system pass):
//   - Component variants gallery (button / pill / chip styles)
//   - Motion / elevation / shadow tokens — none defined yet
//   - Layout density tokens — out of v0
import SwiftUI

struct TokensPreviewView: View {
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Spacing.xxl) {
                colorsSection
                spacingSection
                radiusSection
                typographySection
            }
            .padding(Spacing.l)
        }
        .background(Theme.bgPage.color)
        .navigationTitle("Design tokens")
        .navigationBarTitleDisplayMode(.inline)
    }

    // MARK: - Sections

    @ViewBuilder
    private var colorsSection: some View {
        sectionHeader("Color")
        VStack(spacing: Spacing.s) {
            colorRow("brandDark",       Theme.brandDark)
            colorRow("brandLight",      Theme.brandLight)
            colorRow("brandPale",       Theme.brandPale)
            divider
            colorRow("textPrimary",     Theme.textPrimary)
            colorRow("textSecondary",   Theme.textSecondary)
            divider
            colorRow("bgPage",          Theme.bgPage)
            colorRow("bgCard",          Theme.bgCard)
            colorRow("bgInput",         Theme.bgInput)
            colorRow("borderDefault",   Theme.borderDefault)
            divider
            colorRow("semanticSuccess", Theme.semanticSuccess)
            colorRow("semanticWarning", Theme.semanticWarning)
            colorRow("semanticDanger",  Theme.semanticDanger)
            colorRow("semanticInfo",    Theme.semanticInfo)
        }
    }

    @ViewBuilder
    private var spacingSection: some View {
        sectionHeader("Spacing")
        VStack(alignment: .leading, spacing: Spacing.s) {
            spacingRow("xs",   Spacing.xs)
            spacingRow("s",    Spacing.s)
            spacingRow("m",    Spacing.m)
            spacingRow("l",    Spacing.l)
            spacingRow("xl",   Spacing.xl)
            spacingRow("xxl",  Spacing.xxl)
            spacingRow("xxxl", Spacing.xxxl)
            spacingRow("huge", Spacing.huge)
        }
    }

    @ViewBuilder
    private var radiusSection: some View {
        sectionHeader("Radius")
        HStack(alignment: .bottom, spacing: Spacing.m) {
            radiusSwatch("s",    Radius.s)
            radiusSwatch("m",    Radius.m)
            radiusSwatch("l",    Radius.l)
            radiusSwatch("xl",   Radius.xl)
            radiusSwatch("pill", Radius.pill)
        }
    }

    @ViewBuilder
    private var typographySection: some View {
        sectionHeader("Typography")
        VStack(alignment: .leading, spacing: Spacing.s) {
            typeRow("display",      Typography.display)
            typeRow("title",        Typography.title)
            typeRow("heading",      Typography.heading)
            typeRow("body",         Typography.body)
            typeRow("bodyEmphasis", Typography.bodyEmphasis)
            typeRow("caption",      Typography.caption)
            typeRow("label",        Typography.label)
            typeRow("mono",         Typography.mono)
        }
    }

    // MARK: - Building blocks

    @ViewBuilder
    private func sectionHeader(_ title: String) -> some View {
        Text(title)
            .appFont(Typography.heading)
            .foregroundStyle(Theme.textPrimary.color)
            .padding(.bottom, Spacing.xs)
    }

    private var divider: some View {
        Rectangle()
            .fill(Theme.borderDefault.color)
            .frame(height: 0.5)
            .padding(.vertical, Spacing.xs)
    }

    @ViewBuilder
    private func colorRow(_ name: String, _ pair: ColorPair) -> some View {
        HStack(spacing: Spacing.m) {
            RoundedRectangle(cornerRadius: Radius.s)
                .fill(pair.color)
                .frame(width: 36, height: 36)
                .overlay(
                    RoundedRectangle(cornerRadius: Radius.s)
                        .strokeBorder(Theme.borderDefault.color, lineWidth: 0.5)
                )
            VStack(alignment: .leading, spacing: 0) {
                Text(name)
                    .appFont(Typography.bodyEmphasis)
                    .foregroundStyle(Theme.textPrimary.color)
                Text("light \(pair.light) · dark \(pair.dark)")
                    .appFont(Typography.caption)
                    .foregroundStyle(Theme.textSecondary.color)
            }
            Spacer()
        }
    }

    @ViewBuilder
    private func spacingRow(_ name: String, _ value: CGFloat) -> some View {
        HStack(spacing: Spacing.m) {
            Text(name)
                .appFont(Typography.bodyEmphasis)
                .foregroundStyle(Theme.textPrimary.color)
                .frame(width: 48, alignment: .leading)
            Rectangle()
                .fill(Theme.brandDark.color)
                .frame(width: value, height: 18)
            Text("\(Int(value))pt")
                .appFont(Typography.caption)
                .foregroundStyle(Theme.textSecondary.color)
        }
    }

    @ViewBuilder
    private func radiusSwatch(_ name: String, _ value: CGFloat) -> some View {
        VStack(spacing: Spacing.xs) {
            RoundedRectangle(cornerRadius: min(value, 28))
                .fill(Theme.brandLight.color)
                .frame(width: 56, height: 56)
            Text(name)
                .appFont(Typography.caption)
                .foregroundStyle(Theme.textSecondary.color)
        }
    }

    @ViewBuilder
    private func typeRow(_ name: String, _ font: Font) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("The quick brown fox")
                .font(font)
                .foregroundStyle(Theme.textPrimary.color)
            Text(name)
                .appFont(Typography.caption)
                .foregroundStyle(Theme.textSecondary.color)
        }
    }
}

#Preview {
    NavigationStack {
        TokensPreviewView()
    }
}
