// RuntimeViewFixtureTest — D3.c.4 instrumented assertion that
// RuntimeView actually loads + evaluates a JS bundle on a real
// emulator and that the registered "RuntimeRoot" component renders.
//
// Flow:
//
//   1. @Before: SoLoader.init (needed by RN native libs to dlopen
//      hermesvm + reactnativejni); extract RuntimeRoot.jsbundle from
//      the test APK's assets into a real on-disk file (RuntimeView
//      requires file:// URLs).
//
//   2. @Test: launch FixtureActivity with the bundle path; the
//      activity mounts a RuntimeView and calls loadBundle.  Then poll
//      the UI Automator accessibility tree for the greeting text
//      "Hello from D3.c" — appears once metro-evaluation completes
//      and the React render commits.
//
//   3. @After: scenario.close() tears down the activity, which calls
//      RuntimeView.reset() through the activity lifecycle and
//      releases the per-instance ReactHostImpl.
//
// The 30s timeout swallows emulator cold-start latency (Hermes
// initialization on first bundle load) and CI noise.  Locally on a
// warm emulator it returns in <2s.
package com.appunvs.runtimesdk.test

import android.content.Intent
import androidx.test.core.app.ActivityScenario
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.uiautomator.By
import androidx.test.uiautomator.UiDevice
import androidx.test.uiautomator.Until
import com.facebook.soloader.SoLoader
import org.junit.After
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import java.io.File

@RunWith(AndroidJUnit4::class)
class RuntimeViewFixtureTest {

    private lateinit var bundleFile: File
    private var scenario: ActivityScenario<FixtureActivity>? = null

    @Before
    fun setUp() {
        // RN's native libs (hermesvm, reactnativejni) require SoLoader
        // initialization before the first ReactHostImpl is constructed.
        // The host app's Application normally does this; we're not
        // running through one, so call directly.  targetContext is the
        // module-under-test's context, where the native libs ship.
        val target = InstrumentationRegistry.getInstrumentation().targetContext
        SoLoader.init(target, /* native exopackage = */ false)

        // RuntimeView only accepts file:// URLs; extract the bundle from
        // the test APK's assets into a real cacheDir file.
        val testCtx = InstrumentationRegistry.getInstrumentation().context
        bundleFile = File(testCtx.cacheDir, "RuntimeRoot.jsbundle")
        testCtx.assets.open("RuntimeRoot.jsbundle").use { input ->
            bundleFile.outputStream().use { output -> input.copyTo(output) }
        }
        assertTrue("fixture bundle was not extracted", bundleFile.exists() && bundleFile.length() > 0)
    }

    @After
    fun tearDown() {
        scenario?.close()
        scenario = null
    }

    @Test
    fun runtimeViewRendersFixtureGreeting() {
        val target = InstrumentationRegistry.getInstrumentation().targetContext
        val intent = Intent(target, FixtureActivity::class.java).apply {
            putExtra(FixtureActivity.EXTRA_BUNDLE_PATH, bundleFile.absolutePath)
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        scenario = ActivityScenario.launch(intent)

        // Poll the system-wide accessibility tree for the greeting Text.
        // RN's <Text> renders as a TextView whose accessibility label =
        // its rendered text content.
        val device = UiDevice.getInstance(InstrumentationRegistry.getInstrumentation())
        val found = device.wait(Until.hasObject(By.text("Hello from D3.c")), 30_000)
        assertTrue("greeting text 'Hello from D3.c' not visible after 30s", found)

        val node = device.findObject(By.text("Hello from D3.c"))
        assertNotNull("findObject returned null for greeting", node)
    }
}
