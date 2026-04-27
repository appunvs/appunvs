// AppDelegate — minimal host app for the D3.c.4 iOS instrumented test.
//
// On launch, builds a single UIWindow with a UIViewController whose
// view is a RuntimeView (from RuntimeSDK).  The RuntimeView immediately
// loadBundle()s `RuntimeRoot.jsbundle` from the app bundle resources;
// once metro-evaluation completes, the registered "RuntimeRoot"
// component renders the "Hello from D3.c" greeting.  XCUIApplication
// in RuntimeFixtureUITests asserts that text becomes visible.
//
// No tab bar, no navigation, no extra chrome — the simpler this app
// is, the fewer things can interfere with the UI test's accessibility
// queries.
import UIKit
import RuntimeSDK

@main
final class AppDelegate: UIResponder, UIApplicationDelegate {
    var window: UIWindow?

    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil
    ) -> Bool {
        let window = UIWindow(frame: UIScreen.main.bounds)

        let host = UIViewController()
        host.view.backgroundColor = .black

        let runtime = RuntimeView(frame: host.view.bounds)
        runtime.translatesAutoresizingMaskIntoConstraints = false
        host.view.addSubview(runtime)
        NSLayoutConstraint.activate([
            runtime.topAnchor.constraint(equalTo: host.view.topAnchor),
            runtime.leftAnchor.constraint(equalTo: host.view.leftAnchor),
            runtime.rightAnchor.constraint(equalTo: host.view.rightAnchor),
            runtime.bottomAnchor.constraint(equalTo: host.view.bottomAnchor),
        ])

        if let url = Bundle.main.url(forResource: "RuntimeRoot", withExtension: "jsbundle") {
            // 2-arg overload (no identity) — matches what host shells
            // use when they don't have BoxIdentity threaded through yet.
            runtime.loadBundle(at: url, completion: nil)
        } else {
            // Make the failure mode visible to the UI test: a plain
            // UILabel whose text the assertion will NOT find, so the
            // test fails with a useful message.
            let missing = UILabel(frame: host.view.bounds)
            missing.text = "FIXTURE BUNDLE MISSING — build-fixture.sh did not run before xcodebuild"
            missing.numberOfLines = 0
            missing.textColor = .white
            missing.textAlignment = .center
            missing.accessibilityIdentifier = "fixture-missing"
            host.view.addSubview(missing)
        }

        window.rootViewController = host
        window.makeKeyAndVisible()
        self.window = window
        return true
    }
}
