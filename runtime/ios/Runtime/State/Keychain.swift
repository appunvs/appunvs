// Keychain — minimal generic-password wrapper.  We don't need kSec
// access groups or biometric prompts; the goal is simply "don't put
// auth tokens in UserDefaults plaintext."  Items are scoped to the app
// bundle id automatically.
import Foundation
import Security

enum Keychain {
    static func set(_ value: String, for key: String) {
        let data = Data(value.utf8)
        // Delete any existing item first so we don't have to branch on
        // SecItemUpdate vs SecItemAdd.
        let q: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
        ]
        SecItemDelete(q as CFDictionary)
        var add = q
        add[kSecValueData as String] = data
        add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlock
        SecItemAdd(add as CFDictionary, nil)
    }

    static func get(_ key: String) -> String? {
        let q: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var out: AnyObject?
        let status = SecItemCopyMatching(q as CFDictionary, &out)
        guard status == errSecSuccess, let data = out as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    static func remove(_ key: String) {
        let q: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
        ]
        SecItemDelete(q as CFDictionary)
    }
}
