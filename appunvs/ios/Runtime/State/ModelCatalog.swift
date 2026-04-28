// ModelCatalog — the static list of AI models the chat picker shows.
//
// v0 ships only DeepSeek Chat (the relay's default OpenAIEngine target);
// the picker UI is fully wired so adding GPT / Claude / Gemini later
// means appending an entry here.  Each entry's `id` is the value we
// send as `model` on POST /ai/turn — see /docs/providers.md for the
// canonical id table per provider.
//
// Empty `id` is reserved for "engine default" (i.e. don't send a model
// override; the relay uses whatever cfg.AI.Model is set to).
import Foundation

struct ChatModel: Identifiable, Hashable {
    let id: String       // empty string = engine default
    let name: String     // user-facing label, e.g. "DeepSeek Chat"
    let provider: String // "DeepSeek" / "Anthropic" / etc; subtitle in the picker
}

enum ModelCatalog {
    /// User-selectable models.  v0: a single DeepSeek entry.  Adding a new
    /// model here is the only change needed to surface it in the picker;
    /// the relay's engine validates the id at run time.
    static let all: [ChatModel] = [
        ChatModel(id: "deepseek-chat", name: "DeepSeek Chat", provider: "DeepSeek"),
    ]

    /// First entry is the catalog default — used when AppState.defaultModel
    /// is unset (fresh install) or names a model that's been removed.
    static var fallback: ChatModel { all.first! }

    static func find(id: String) -> ChatModel? {
        all.first(where: { $0.id == id })
    }
}
