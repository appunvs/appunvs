// AppunvsHostPackage — registers AppunvsHostModule with a ReactHost.
//
// RuntimeView.kt creates each ReactHost via DefaultReactHostDelegate;
// the delegate's `reactPackages` list is what the RN bootstrap walks
// to discover native modules.  D3.c.2 used `emptyList()` because we
// hadn't built any native modules yet; D3.e.1 includes this single
// package so AI bundles can `import` `@appunvs/host` with something
// real on the other side.
package com.appunvs.runtimesdk

import com.facebook.react.ReactPackage
import com.facebook.react.bridge.NativeModule
import com.facebook.react.bridge.ReactApplicationContext
import com.facebook.react.uimanager.ViewManager

class AppunvsHostPackage(
    private val identity: RuntimeBoxIdentity = RuntimeBoxIdentity.EMPTY,
) : ReactPackage {

    override fun createNativeModules(
        reactContext: ReactApplicationContext,
    ): List<NativeModule> = listOf(AppunvsHostModule(reactContext, identity))

    override fun createViewManagers(
        reactContext: ReactApplicationContext,
    ): List<ViewManager<*, *>> = emptyList()
}
