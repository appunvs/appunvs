// RuntimeBoxIdentity — read-only descriptor of the AI-generated Box
// the host has asked us to load.  Mirrors `BoxIdentity` in
// runtime/src/HostBridge.ts so AI bundles see exactly this shape via
// `host().identity`.
//
// Passed to RuntimeView.loadBundle(url, identity, completion) at load
// time.  The runtime SDK doesn't validate any field — the host shell
// is responsible for sourcing them from its relay client (BoxWire.id /
// version / title).
package com.appunvs.runtimesdk

data class RuntimeBoxIdentity(
    /** Stable per-Box id assigned by the relay.  Empty string on dev /
     *  unbuilt drafts.
     */
    val boxID: String,

    /** Bundle version string (e.g. "v3").  Empty string on unbuilt drafts. */
    val version: String,

    /** Short title for display, mirrors BoxWire.title from the relay. */
    val title: String,
) {
    companion object {
        @JvmField
        val EMPTY = RuntimeBoxIdentity(boxID = "", version = "", title = "")
    }
}
