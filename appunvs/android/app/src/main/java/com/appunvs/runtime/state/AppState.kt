// AppState — host-wide observable state held in a ViewModel.
// Currently carries the theme override (system / light / dark).
// DataStore Preferences provides a small async key-value store; we
// load on init and persist on every override change.
//
// Extends naturally to active Box reference, auth tokens, relay
// connection status as those slices land.
package com.appunvs.runtime.state

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.firstOrNull
import kotlinx.coroutines.launch

/// Module-level DataStore handle.  Names the on-disk file that backs
/// our Preferences; keep distinct from any other DataStore the app
/// might add later.
private val Application.dataStore by preferencesDataStore(name = "appunvs.prefs")

/// AndroidViewModel because we need an Application reference for the
/// DataStore handle.  Compose calls `viewModel()` and gets the same
/// instance for the lifetime of the activity.
class AppState(application: Application) : AndroidViewModel(application) {

    enum class ThemeOverride { SYSTEM, LIGHT, DARK }

    private val themeKey = stringPreferencesKey("theme.override")
    private val modelKey = stringPreferencesKey("model.default")
    private val store: androidx.datastore.core.DataStore<Preferences> = application.dataStore

    /// Compose-observable mirror of the persisted override.  Updated on
    /// init from disk, and on every set call (which also persists).
    var themeOverride by mutableStateOf(ThemeOverride.SYSTEM)
        private set

    /// Default chat model id (e.g. "deepseek-chat") — what the composer
    /// uses when the user hasn't set a per-turn override on the chat
    /// header.  Persisted via DataStore.  Unknown ids fall back to
    /// ModelCatalog.fallback.id so a release that drops a model doesn't
    /// strand existing users.
    var defaultModelID by mutableStateOf(ModelCatalog.fallback.id)
        private set

    init {
        viewModelScope.launch {
            val prefs = store.data.firstOrNull()
            themeOverride = parse(prefs?.get(themeKey))
            val raw = prefs?.get(modelKey).orEmpty()
            defaultModelID = if (raw.isNotEmpty() && ModelCatalog.find(raw) != null) {
                raw
            } else {
                ModelCatalog.fallback.id
            }
        }
    }

    fun setTheme(value: ThemeOverride) {
        themeOverride = value
        viewModelScope.launch {
            store.edit { prefs -> prefs[themeKey] = value.name }
        }
    }

    fun setDefaultModel(id: String) {
        defaultModelID = id
        viewModelScope.launch {
            store.edit { prefs -> prefs[modelKey] = id }
        }
    }

    private fun parse(raw: String?): ThemeOverride =
        when (raw) {
            ThemeOverride.LIGHT.name -> ThemeOverride.LIGHT
            ThemeOverride.DARK.name  -> ThemeOverride.DARK
            else                     -> ThemeOverride.SYSTEM
        }
}
