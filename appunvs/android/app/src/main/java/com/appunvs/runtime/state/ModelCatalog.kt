// ModelCatalog — the static list of AI models the chat picker shows.
//
// v0 ships only DeepSeek Chat (the relay's default OpenAIEngine target);
// the picker UI is fully wired so adding GPT / Claude / Gemini later
// means appending an entry here.  Each entry's `id` is the value sent
// as `model` on POST /ai/turn — see /docs/providers.md for the
// canonical id table per provider.  Mirrors iOS ModelCatalog.swift.
package com.appunvs.runtime.state

data class ChatModel(
    val id: String,       // empty = engine default
    val name: String,     // user-facing label, e.g. "DeepSeek Chat"
    val provider: String, // "DeepSeek" / "Anthropic" / etc.
)

object ModelCatalog {
    /// User-selectable models.  v0: a single DeepSeek entry.  Add one
    /// here and it surfaces in both the Profile default picker and the
    /// Chat header chip with no further changes.
    val all: List<ChatModel> = listOf(
        ChatModel(id = "deepseek-chat", name = "DeepSeek Chat", provider = "DeepSeek"),
    )

    /// First entry is the catalog default — used when AppState's stored
    /// id is unset (fresh install) or names a model that's been removed.
    val fallback: ChatModel get() = all.first()

    fun find(id: String): ChatModel? = all.firstOrNull { it.id == id }
}
