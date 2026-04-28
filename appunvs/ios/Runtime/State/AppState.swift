// AppState — host-wide observable state.  Currently carries the user's
// theme override (light / dark / nil = follow system).  Persists to
// UserDefaults so the choice survives restarts.
//
// Extends naturally to hold the active Box reference, the relay
// connection status, and the auth tokens once the network slice lands.
import Foundation
import SwiftUI
import Combine

@MainActor
final class AppState: ObservableObject {

    enum ThemeOverride: String, CaseIterable, Identifiable {
        case system, light, dark
        var id: String { rawValue }
    }

    /// User-selected theme override.  `system` defers to OS Dynamic Type
    /// trait; `light` / `dark` force the corresponding scheme.
    @Published var themeOverride: ThemeOverride {
        didSet { Self.persistTheme(themeOverride) }
    }

    /// Default model id used by the chat composer when the user hasn't
    /// picked a per-turn override.  Stored as the raw provider id (e.g.
    /// "deepseek-chat"); ModelCatalog.find resolves it to a display
    /// entry.  Empty string means "use the relay's engine default".
    @Published var defaultModelID: String {
        didSet { Self.persistModel(defaultModelID) }
    }

    /// Resolved value handed to `.preferredColorScheme(...)` on the root
    /// view.  `nil` means "let the system decide".
    var preferredColorScheme: ColorScheme? {
        switch themeOverride {
        case .system: return nil
        case .light:  return .light
        case .dark:   return .dark
        }
    }

    init() {
        themeOverride = Self.loadTheme()
        defaultModelID = Self.loadModel()
    }

    // MARK: - persistence

    private static let themeKey = "appunvs.theme.override"
    private static let modelKey = "appunvs.model.default"

    private static func loadTheme() -> ThemeOverride {
        guard let raw = UserDefaults.standard.string(forKey: themeKey),
              let parsed = ThemeOverride(rawValue: raw)
        else { return .system }
        return parsed
    }

    private static func persistTheme(_ value: ThemeOverride) {
        UserDefaults.standard.set(value.rawValue, forKey: themeKey)
    }

    private static func loadModel() -> String {
        // Validate against the catalog so a release that drops an old id
        // doesn't leave the user pinned to a model the relay no longer
        // accepts; fall back to the first catalog entry instead.
        let raw = UserDefaults.standard.string(forKey: modelKey) ?? ""
        if raw.isEmpty { return ModelCatalog.fallback.id }
        if ModelCatalog.find(id: raw) != nil { return raw }
        return ModelCatalog.fallback.id
    }

    private static func persistModel(_ value: String) {
        UserDefaults.standard.set(value, forKey: modelKey)
    }
}
