// SecureStore — minimal wrapper around EncryptedSharedPreferences.  We
// don't need biometric prompts; the goal is "don't put auth tokens in
// SharedPreferences plaintext."  The master key is per-app-install,
// stored in the Android Keystore.
package com.appunvs.runtime.state

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

object SecureStore {
    private const val FILE = "appunvs.secure"

    fun open(context: Context): SharedPreferences {
        val key = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        return EncryptedSharedPreferences.create(
            context,
            FILE,
            key,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }
}
