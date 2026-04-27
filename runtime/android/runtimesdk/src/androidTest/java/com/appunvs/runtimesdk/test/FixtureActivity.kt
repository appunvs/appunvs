// FixtureActivity — minimal Activity used by the instrumented UI test.
//
// Created on demand by the test (ActivityScenario.launch with an
// Intent carrying the bundle path), it just mounts a RuntimeView as
// its content view and calls loadBundle.  The test then asserts via
// UI Automator that the fixture's "Hello from D3.c" greeting is
// visible — proving the SDK loaded + evaluated + rendered the bundle.
package com.appunvs.runtimesdk.test

import android.app.Activity
import android.os.Bundle
import com.appunvs.runtimesdk.RuntimeView

class FixtureActivity : Activity() {

    companion object {
        /// Intent extra: absolute filesystem path to the JS bundle the
        /// test extracted from its assets into the test app's cacheDir.
        /// Required — the activity errors out fast if missing.
        const val EXTRA_BUNDLE_PATH = "bundle_path"
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val path = intent.getStringExtra(EXTRA_BUNDLE_PATH)
            ?: error("FixtureActivity launched without $EXTRA_BUNDLE_PATH extra")

        val view = RuntimeView(this)
        setContentView(view)
        view.loadBundle("file://$path", null)
    }
}
