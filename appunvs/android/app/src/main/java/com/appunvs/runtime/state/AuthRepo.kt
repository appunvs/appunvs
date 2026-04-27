// AuthRepo — owns the local view of the user's auth state, mirroring
// the iOS AuthStore.  An AndroidViewModel so it can pull a Context for
// EncryptedSharedPreferences (token storage) and DataStore-equivalent
// per-install device id.
//
// Lifecycle:
//   1. boot()       loads persisted device token (if any), drives gate
//   2. signup/login mints a session token, then immediately calls
//                  /auth/register to swap it for a device token
//   3. signOut()    clears EncryptedSharedPrefs + memory
package com.appunvs.runtime.state

import android.app.Application
import android.content.SharedPreferences
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.appunvs.runtime.net.AISSEClient
import com.appunvs.runtime.net.LoginRequest
import com.appunvs.runtime.net.MeResponse
import com.appunvs.runtime.net.RegisterRequest
import com.appunvs.runtime.net.RelayApi
import com.appunvs.runtime.net.RelayBundle
import com.appunvs.runtime.net.RelayClient
import okhttp3.OkHttpClient
import com.appunvs.runtime.net.SignupRequest
import com.appunvs.runtime.net.TokenSource
import kotlinx.coroutines.launch
import java.util.UUID

class AuthRepo(application: Application) : AndroidViewModel(application) {

    sealed class Phase {
        data object Bootstrapping : Phase()
        data object SignedOut    : Phase()
        data class  SignedIn(val userID: String) : Phase()
    }

    private val secure: SharedPreferences = SecureStore.open(application)

    /// Active token surfaced to the OkHttp interceptor.  Briefly holds
    /// the session token during signup / login (so /auth/register works);
    /// then carries the long-lived device token for everything else.
    @Volatile
    private var token: String? = null
    private val tokenSource = TokenSource { token }

    private val bundle: RelayBundle = RelayClient.build(tokenSource)
    private val api: RelayApi = bundle.api
    val sse: AISSEClient = AISSEClient(bundle.http)

    /// Auth-interceptor-wrapped OkHttpClient — exposed so the SDK
    /// bridge wiring (RuntimeBridgeWiring) can issue arbitrary requests
    /// from AI bundles via host().network.request without re-implementing
    /// the auth header injection.  Token rotation is automatic since
    /// the interceptor re-reads tokenSource on every call.
    internal fun http(): OkHttpClient = bundle.http

    var phase by mutableStateOf<Phase>(Phase.Bootstrapping)
        private set

    var me by mutableStateOf<MeResponse?>(null)
        private set

    var lastError by mutableStateOf<String?>(null)

    init {
        viewModelScope.launch { boot() }
    }

    private fun boot() {
        val saved = secure.getString(KEY_TOKEN, null)
        val userID = secure.getString(KEY_USER_ID, null)
        if (saved != null && userID != null) {
            token = saved
            phase = Phase.SignedIn(userID)
        } else {
            phase = Phase.SignedOut
        }
    }

    fun signup(email: String, password: String) {
        viewModelScope.launch { runAuth("signup") { api.signup(SignupRequest(email, password)) } }
    }

    fun login(email: String, password: String) {
        viewModelScope.launch { runAuth("login") { api.login(LoginRequest(email, password)) } }
    }

    fun signOut() {
        token = null
        me = null
        secure.edit().remove(KEY_TOKEN).remove(KEY_USER_ID).apply()
        phase = Phase.SignedOut
    }

    private suspend fun runAuth(label: String, op: suspend () -> com.appunvs.runtime.net.SessionResponse) {
        lastError = null
        try {
            val session = op()
            // 1. Hold the session token transiently so /auth/register
            //    accepts it.
            token = session.sessionToken
            // 2. Capture profile while the session token is still hot.
            //    Failure here is non-fatal — proceed to /register.
            me = runCatching { api.me() }.getOrNull()
            // 3. Swap to a device token for the rest of the session.
            val dev = api.registerDevice(
                RegisterRequest(deviceID = stableDeviceID(), platform = "mobile"),
            )
            token = dev.token
            secure.edit()
                .putString(KEY_TOKEN, dev.token)
                .putString(KEY_USER_ID, session.userID)
                .apply()
            phase = Phase.SignedIn(session.userID)
        } catch (t: Throwable) {
            lastError = "$label failed: ${t.message ?: t.javaClass.simpleName}"
            token = null
        }
    }

    /// A per-install identifier reused across launches so the relay
    /// keeps one device row per emulator/device.  Persists in the same
    /// EncryptedSharedPreferences file (cheap; survives signout).
    private fun stableDeviceID(): String {
        val existing = secure.getString(KEY_DEVICE_ID, null)
        if (existing != null) return existing
        val id = "d_" + UUID.randomUUID().toString().replace("-", "").lowercase()
        secure.edit().putString(KEY_DEVICE_ID, id).apply()
        return id
    }

    /// Exposed so BoxRepo / ChatViewModel can share the same Retrofit
    /// instance — they're scoped to the same auth session.
    internal fun api(): RelayApi = api

    private companion object {
        const val KEY_TOKEN     = "deviceToken"
        const val KEY_USER_ID   = "userId"
        const val KEY_DEVICE_ID = "deviceId"
    }
}
