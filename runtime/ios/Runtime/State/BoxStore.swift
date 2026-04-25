// BoxStore — owns the box list + the active box.  Backed by real
// /box endpoint calls.
//
// `refresh()` is fire-and-forget from the UI; errors surface via
// `lastError` so the Chat / Stage screens can render an inline retry
// affordance later.
import Foundation
import SwiftUI

@MainActor
final class BoxStore: ObservableObject {
    @Published private(set) var boxes: [BoxWire] = []
    @Published var activeBox: BoxWire?
    @Published var lastError: String?
    @Published private(set) var loading: Bool = false

    private var api: BoxAPI

    /// `nonisolated` so the placeholder construction in
    /// `SignedInRoot.init` (a non-isolated synchronous View init under
    /// `@preconcurrency View`) doesn't need to hop the main actor just
    /// to assign one stored property.
    nonisolated init(http: HTTPClient) {
        self.api = BoxAPI(http: http)
    }

    /// Swap the underlying HTTP client.  Used by RuntimeApp to replace
    /// the StateObject's placeholder client with the AuthStore-backed one
    /// once the user is signed in.
    func rebind(http: HTTPClient) {
        self.api = BoxAPI(http: http)
    }

    func refresh() async {
        loading = true
        defer { loading = false }
        do {
            let next = try await api.list()
            self.boxes = next
            // Preserve active selection by id; default to first box if
            // the previous selection is gone.
            if let cur = activeBox, let still = next.first(where: { $0.boxID == cur.boxID }) {
                activeBox = still
            } else {
                activeBox = next.first
            }
        } catch let err as HTTPError {
            lastError = err.errorDescription
        } catch {
            lastError = error.localizedDescription
        }
    }

    func setActive(_ box: BoxWire) {
        activeBox = box
    }

    func create(title: String) async {
        do {
            let resp = try await api.create(title: title)
            boxes.insert(resp.box, at: 0)
            activeBox = resp.box
        } catch let err as HTTPError {
            lastError = err.errorDescription
        } catch {
            lastError = error.localizedDescription
        }
    }
}
