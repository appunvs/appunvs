// RuntimeView — D3.c implementation: real React Native mount.
//
// Per-instance `ReactHostImpl` constructed directly (NOT via the cached
// `DefaultReactHost.getDefaultReactHost`) so each RuntimeView runs an
// isolated Hermes runtime — `reset` releases the host, `loadBundle` on
// a new URL boots a fresh one.  Cross-bundle JS state cannot leak.
//
// JS contract: the bundle MUST register a component named "RuntimeRoot"
// via `AppRegistry.registerComponent`.  Same shape as the iOS side.
//
// URL forms accepted:
//   - file:///abs/path  → loaded via JSBundleLoader.createFileLoader
//
// Network URLs (http/https) are NOT accepted yet — D3.e adds a
// fetch-and-cache layer.  Host apps should download the bundle to a
// cache file, then pass file:// here.
package com.appunvs.runtimesdk

import android.app.Application
import android.content.Context
import android.graphics.Color
import android.net.Uri
import android.util.AttributeSet
import android.widget.FrameLayout
import com.facebook.react.bridge.JSBundleLoader
import com.facebook.react.common.annotations.UnstableReactNativeAPI
import com.facebook.react.defaults.DefaultComponentsRegistry
import com.facebook.react.defaults.DefaultReactHostDelegate
import com.facebook.react.defaults.DefaultTurboModuleManagerDelegate
import com.facebook.react.fabric.ComponentFactory
import com.facebook.react.runtime.ReactHostImpl

class RuntimeView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
    defStyleAttr: Int = 0,
) : FrameLayout(context, attrs, defStyleAttr) {

    /** The bundle URL currently displayed (or being loaded).  null
     *  before the first loadBundle call.
     */
    var currentBundleURL: String? = null
        private set

    /** The Box identity currently mounted.  null before the first
     *  identity-bearing loadBundle call.
     */
    var currentIdentity: RuntimeBoxIdentity? = null
        private set

    private var reactHost: ReactHostImpl? = null

    init {
        setBackgroundColor(Color.BLACK)
    }

    /**
     * Asks the runtime to fetch the bundle at [url] and mount its
     * React tree into this view's bounds.  [identity] is exposed to
     * the JS runtime as `host().identity` (boxID / version / title).
     * Calling this while another bundle is loaded resets first.
     *
     * [url] must be a `file://` URL pointing to a downloaded JS bundle
     * on disk.  D3.e will add direct http(s) support.
     *
     * [completion] is invoked on the main thread (`post()`) — null on
     * success.  Bundle-load errors surface via RN's red-box overlay
     * inside the mounted view; D3.e wires a real progress / error path.
     */
    /** Backwards-compat overload — host call sites that don't yet
     *  carry identity continue to compile.  Forwards to the
     *  identity-bearing variant with [RuntimeBoxIdentity.EMPTY].
     */
    fun loadBundle(url: String, completion: ((Throwable?) -> Unit)?) {
        loadBundle(url, RuntimeBoxIdentity.EMPTY, completion)
    }

    @OptIn(UnstableReactNativeAPI::class)
    @JvmOverloads
    fun loadBundle(
        url: String,
        identity: RuntimeBoxIdentity = RuntimeBoxIdentity.EMPTY,
        completion: ((Throwable?) -> Unit)? = null,
    ) {
        reset()

        currentBundleURL = url
        currentIdentity = identity

        val parsed = Uri.parse(url)
        require(parsed.scheme == "file") {
            "RuntimeView.loadBundle() only accepts file:// URLs in D3.c " +
                "(got scheme '${parsed.scheme}'); the host should download " +
                "the bundle to disk first.  D3.e adds a fetch-and-cache layer."
        }
        val filePath = requireNotNull(parsed.path) {
            "file:// URL has no path component: $url"
        }

        val application = context.applicationContext as Application
        val componentFactory = ComponentFactory().also {
            DefaultComponentsRegistry.register(it)
        }

        // Per-instance bundle loader → per-instance Hermes runtime.
        // No reactPackages: D3.c keeps the SDK runtime clean of native
        // modules.  D3.d threads in the Tier 1 set (gesture-handler /
        // reanimated / etc.).
        val delegate = DefaultReactHostDelegate(
            jsMainModulePath = "index",
            jsBundleLoader = JSBundleLoader.createFileLoader(filePath),
            // D3.e.1: AppunvsHostPackage is the single SDK-owned package
            // — exposes `NativeModules.AppunvsHost` to JS so AI bundles'
            // `@appunvs/host` imports have a real native target.  Tier 1
            // RN packages (reanimated etc.) auto-register at runtime via
            // their AAR's manifest entries; we don't list them here.
            // D3.e.2: identity propagates via the package's constructor
            // — each per-RuntimeView ReactHost gets its own pinned-to-
            // this-Box AppunvsHostModule.
            reactPackages = listOf(AppunvsHostPackage(identity)),
            // Required as of RN 0.85.2 (the no-arg default was removed).
            // Empty builder is fine because reactPackages is empty too —
            // there are no turbo modules to register.
            turboModuleManagerDelegateBuilder = DefaultTurboModuleManagerDelegate.Builder(),
        )
        val host = ReactHostImpl(
            application,
            delegate,
            componentFactory,
            /* allowPackagerServerAccess = */ false,
            /* useDevSupport = */ false,
        )
        reactHost = host

        val surface = host.createSurface(context, "RuntimeRoot", null)
        addView(
            surface.view,
            LayoutParams(LayoutParams.MATCH_PARENT, LayoutParams.MATCH_PARENT),
        )
        surface.start()

        post { completion?.invoke(null) }
    }

    /** Tears down the current bundle's runtime state.  No-op when
     *  no bundle is loaded.
     */
    fun reset() {
        removeAllViews()
        reactHost?.destroy("RuntimeView.reset", null)
        reactHost = null
        currentBundleURL = null
        currentIdentity = null
    }
}
