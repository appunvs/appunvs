// Theme — design tokens carried over from the prior Expo prototype's
// `app/src/theme/colors.ts`.  Each token here is a `ColorPair` (light +
// dark hex strings); SwiftUI consumers reach for `.color` which resolves
// against the current trait collection at render time.
//
// We deliberately avoid the asset catalog approach — programmatic
// `UIColor { trait in ... }` keeps the tokens in source code where
// they're greppable and version-controlled inline with the consumer.
import SwiftUI
import UIKit

/// A pair of hex strings (light, dark) that produces a single trait-aware
/// `Color`.  Hex format: "#RRGGBB" or "#AARRGGBB".
struct ColorPair {
    let light: String
    let dark: String

    var color: Color {
        Color(uiColor: UIColor { trait in
            let hex = trait.userInterfaceStyle == .dark ? dark : light
            return UIColor(hex: hex)
        })
    }
}

enum Theme {
    // Brand
    static let brandDark    = ColorPair(light: "#0B505A", dark: "#4FB0BE")
    static let brandLight   = ColorPair(light: "#6FC0CC", dark: "#167C8C")
    static let brandPale    = ColorPair(light: "#E9F4F5", dark: "#14353B")

    // Text
    static let textPrimary   = ColorPair(light: "#152127", dark: "#E8F0F2")
    static let textSecondary = ColorPair(light: "#557280", dark: "#9AB0B8")

    // Surface
    static let bgPage        = ColorPair(light: "#F2F6F6", dark: "#0B1418")
    static let bgCard        = ColorPair(light: "#FFFFFF", dark: "#152127")
    static let bgInput       = ColorPair(light: "#E9EFF0", dark: "#1E2D33")
    static let borderDefault = ColorPair(light: "#DAE4E6", dark: "#243339")

    // Semantic
    static let semanticSuccess = ColorPair(light: "#1F7A4D", dark: "#5BD391")
    static let semanticWarning = ColorPair(light: "#A65A0E", dark: "#F0B45A")
    static let semanticDanger  = ColorPair(light: "#B23A3A", dark: "#F08585")
    static let semanticInfo    = ColorPair(light: "#155E96", dark: "#7CB6E5")
}

/// Spacing scale — index by name at call sites to keep numbers out of
/// view code.  Named scale matches the prior RN tokens.
enum Spacing {
    static let xs:   CGFloat = 4
    static let s:    CGFloat = 8
    static let m:    CGFloat = 12
    static let l:    CGFloat = 16
    static let xl:   CGFloat = 20
    static let xxl:  CGFloat = 24
    static let xxxl: CGFloat = 32
    static let huge: CGFloat = 48
}

/// Corner radius scale.  `xl` is reserved for chat bubbles.
enum Radius {
    static let s:  CGFloat = 6
    static let m:  CGFloat = 10
    static let l:  CGFloat = 12
    static let xl: CGFloat = 14
    static let pill: CGFloat = 999
}

// MARK: - UIColor hex helper

extension UIColor {
    /// Construct from "#RRGGBB" or "#AARRGGBB".  Falls back to magenta
    /// on malformed input — a "this should be obvious in preview" choice
    /// rather than a silent failure.
    convenience init(hex: String) {
        var s = hex
        if s.hasPrefix("#") { s.removeFirst() }
        let scanner = Scanner(string: s)
        var v: UInt64 = 0
        guard scanner.scanHexInt64(&v) else {
            self.init(red: 1, green: 0, blue: 1, alpha: 1)
            return
        }
        let a, r, g, b: CGFloat
        switch s.count {
        case 6:
            a = 1
            r = CGFloat((v >> 16) & 0xFF) / 255
            g = CGFloat((v >> 8)  & 0xFF) / 255
            b = CGFloat( v        & 0xFF) / 255
        case 8:
            a = CGFloat((v >> 24) & 0xFF) / 255
            r = CGFloat((v >> 16) & 0xFF) / 255
            g = CGFloat((v >> 8)  & 0xFF) / 255
            b = CGFloat( v        & 0xFF) / 255
        default:
            self.init(red: 1, green: 0, blue: 1, alpha: 1)
            return
        }
        self.init(red: r, green: g, blue: b, alpha: a)
    }
}
