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
        didSet { Self.persist(themeOverride) }
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
        themeOverride = Self.load()
    }

    // MARK: - persistence

    private static let key = "appunvs.theme.override"

    private static func load() -> ThemeOverride {
        guard let raw = UserDefaults.standard.string(forKey: key),
              let parsed = ThemeOverride(rawValue: raw)
        else { return .system }
        return parsed
    }

    private static func persist(_ value: ThemeOverride) {
        UserDefaults.standard.set(value.rawValue, forKey: key)
    }
}
