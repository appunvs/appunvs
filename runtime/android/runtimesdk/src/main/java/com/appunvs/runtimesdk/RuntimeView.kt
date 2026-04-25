// RuntimeView — public surface for mounting an AI-generated bundle
// inside the host's Stage tab.
//
// D2.c (this PR): placeholder FrameLayout that just displays the
// loaded bundle URL as a centred TextView — proves the API shape
// end-to-end, callable from host Compose code, observable via
// loadBundle / reset.
//
// D3 replaces the placeholder impl with the real React Native +
// Hermes mount.  The public surface here stays stable so host code
// doesn't move between D2.c and D3.
//
// Forward-declarations for D3:
//
//   - The placeholder's TextView swaps for a ReactRootView (or its
//     bridgeless equivalent) hosted by ReactHost.
//   - loadBundle(url) will fetch the JS bundle, evaluate it under a
//     dedicated Hermes runtime, and mount the React tree.
//   - reset will tear down the Hermes runtime and prepare for a
//     fresh bundle load.
package com.appunvs.runtimesdk

import android.content.Context
import android.graphics.Color
import android.graphics.Typeface
import android.util.AttributeSet
import android.view.Gravity
import android.view.ViewGroup
import android.widget.FrameLayout
import android.widget.TextView

class RuntimeView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
    defStyleAttr: Int = 0,
) : FrameLayout(context, attrs, defStyleAttr) {

    /** The bundle URL currently displayed (or being loaded). null
     *  before the first loadBundle call.
     */
    var currentBundleURL: String? = null
        private set

    private val statusLabel: TextView = TextView(context).apply {
        layoutParams = LayoutParams(
            ViewGroup.LayoutParams.WRAP_CONTENT,
            ViewGroup.LayoutParams.WRAP_CONTENT,
            Gravity.CENTER,
        )
        gravity = Gravity.CENTER
        setTextColor(Color.LTGRAY)
        textSize = 13f
        typeface = Typeface.MONOSPACE
        text = "no bundle loaded"
    }

    init {
        setBackgroundColor(Color.BLACK)
        addView(statusLabel)
    }

    /**
     * Asks the runtime to fetch the bundle at [url] and mount its
     * React tree into this view's bounds.  Calling this while another
     * bundle is loaded resets first.  D2.c placeholder: just stores
     * the URL and displays it; D3 replaces with real Hermes mount.
     *
     * [completion] is invoked on the main thread (post()) — null on
     * success, an exception on failure.  Always called exactly once.
     */
    @JvmOverloads
    fun loadBundle(url: String, completion: ((Throwable?) -> Unit)? = null) {
        reset()

        currentBundleURL = url
        statusLabel.text = "loading\n$url"

        // D2.c placeholder: display the URL on the next event-loop turn.
        post {
            statusLabel.text =
                "loaded\n$url\n\n(D3 replaces this with a real React Native render)"
            completion?.invoke(null)
        }
    }

    /** Tears down the current bundle's runtime state.  No-op when
     *  no bundle is loaded.
     */
    fun reset() {
        currentBundleURL = null
        statusLabel.text = "(reset; no bundle)"
    }
}
