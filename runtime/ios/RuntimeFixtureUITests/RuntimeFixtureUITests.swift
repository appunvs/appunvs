// RuntimeFixtureUITests — D3.c.4 iOS instrumented assertion that
// RuntimeView actually loads + evaluates the fixture bundle and
// renders the registered "RuntimeRoot" component on a real simulator.
//
// XCUITest pattern: launch RuntimeFixtureApp (which mounts a
// RuntimeView and calls loadBundle on RuntimeRoot.jsbundle from its
// own bundle resources), then poll the accessibility tree for the
// fixture's greeting Text.  Times out at 30s — generous to swallow
// simulator cold-start + Hermes initialization.
//
// The 30s mirrors the Android instrumented test's tolerance; locally
// on a warm sim it returns in under 2s.
import XCTest

final class RuntimeFixtureUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testRuntimeViewRendersFixtureGreeting() throws {
        let app = XCUIApplication()
        // Bundle id matches RuntimeFixtureApp's PRODUCT_BUNDLE_IDENTIFIER
        // in SDK.yml.  XCUIApplication() with no args picks the
        // host-app target's bundle id automatically per the test target's
        // TEST_TARGET_NAME setting.
        app.launch()

        // RN's <Text testID="runtime-greeting">Hello from D3.c</Text>
        // surfaces in the iOS accessibility tree as a static text whose
        // identifier (testID) AND label (visible text) we can match on.
        // We match on label here because the fixture is a single static
        // text element — simpler than wiring testID through XCUITest.
        let greeting = app.staticTexts["Hello from D3.c"]
        let appeared = greeting.waitForExistence(timeout: 30)
        XCTAssertTrue(
            appeared,
            "fixture greeting did not render within 30s — RuntimeView " +
            "either failed to load the bundle, failed to evaluate it, " +
            "or the registered RuntimeRoot component did not commit a " +
            "render.  Check the fixture app's load completion path."
        )
    }
}
