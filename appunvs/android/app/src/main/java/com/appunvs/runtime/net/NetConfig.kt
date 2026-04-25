// NetConfig — relay base URL.  Defaults to the Android emulator's host
// bridge (10.0.2.2) so a relay running on the dev's machine at
// localhost:8080 is reachable.  Override at runtime by setting
// `APPUNVS_RELAY_URL` as a system property — useful when pointing a
// real device at a LAN address.
package com.appunvs.runtime.net

object NetConfig {
    val relayBaseURL: String
        get() = System.getProperty("APPUNVS_RELAY_URL")
            ?: "http://10.0.2.2:8080"
}
