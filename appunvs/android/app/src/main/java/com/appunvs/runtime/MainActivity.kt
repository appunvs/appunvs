// MainActivity — single Activity hosting the Compose tree.  Three
// top-level tabs (Chat / Stage / Profile) live inside; nothing else
// routes to its own Activity in this design.
//
// Auth gate:
//   * AuthRepo bootstraps from EncryptedSharedPreferences on init
//   * Phase.Bootstrapping  -> spinner
//   * Phase.SignedOut      -> LoginScreen
//   * Phase.SignedIn       -> RuntimeRoot (tabs)
//
// BoxRepo + ChatViewModel are constructed inside the signed-in branch
// so they share AuthRepo's Retrofit interface / OkHttp client (hence
// the same token rotation surface).
package com.appunvs.runtime

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AccountCircle
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.PlayCircleOutline
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel

import com.appunvs.runtime.screens.ChatScreen
import com.appunvs.runtime.screens.LoginScreen
import com.appunvs.runtime.screens.ProfileScreen
import com.appunvs.runtime.screens.StageScreen
import com.appunvs.runtime.state.AppState
import com.appunvs.runtime.state.AuthRepo
import com.appunvs.runtime.state.BoxRepo
import com.appunvs.runtime.state.ChatViewModel
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.RuntimeTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            Gate()
        }
    }
}

@Composable
private fun Gate(
    state: AppState = viewModel(),
    auth: AuthRepo = viewModel(),
) {
    RuntimeTheme(themeOverride = state.themeOverride) {
        when (val phase = auth.phase) {
            AuthRepo.Phase.Bootstrapping -> BootSplash()
            AuthRepo.Phase.SignedOut     -> LoginScreen(auth = auth, modifier = Modifier.fillMaxSize())
            is AuthRepo.Phase.SignedIn   -> SignedInRoot(state = state, auth = auth)
        }
    }
}

@Composable
private fun BootSplash() {
    val colors = LocalAppColors.current
    Box(
        modifier = Modifier.fillMaxSize().background(colors.bgPage),
        contentAlignment = Alignment.Center,
    ) {
        CircularProgressIndicator(color = colors.brandDark)
    }
}

@Composable
private fun SignedInRoot(
    state: AppState,
    auth: AuthRepo,
) {
    val boxRepo: BoxRepo = viewModel(factory = SimpleFactory { BoxRepo(auth.api()) })
    val chat: ChatViewModel = viewModel(factory = SimpleFactory { ChatViewModel(auth.sse) })

    LaunchedEffect(Unit) { boxRepo.refresh() }

    var selected by remember { mutableStateOf(Tab.CHAT) }
    Scaffold(
        bottomBar = {
            NavigationBar {
                NavigationBarItem(
                    selected = selected == Tab.CHAT,
                    onClick = { selected = Tab.CHAT },
                    icon = { Icon(Icons.Outlined.ChatBubbleOutline, contentDescription = null) },
                    label = { Text("Chat") },
                )
                NavigationBarItem(
                    selected = selected == Tab.STAGE,
                    onClick = { selected = Tab.STAGE },
                    icon = { Icon(Icons.Outlined.PlayCircleOutline, contentDescription = null) },
                    label = { Text("Stage") },
                )
                NavigationBarItem(
                    selected = selected == Tab.PROFILE,
                    onClick = { selected = Tab.PROFILE },
                    icon = { Icon(Icons.Outlined.AccountCircle, contentDescription = null) },
                    label = { Text("Profile") },
                )
            }
        },
    ) { padding ->
        val tabModifier = Modifier.fillMaxSize().padding(padding)
        when (selected) {
            Tab.CHAT    -> ChatScreen(boxRepo = boxRepo, chat = chat, modifier = tabModifier)
            Tab.STAGE   -> StageScreen(boxRepo = boxRepo, modifier = tabModifier)
            Tab.PROFILE -> ProfileScreen(state = state, auth = auth, modifier = tabModifier)
        }
    }
}

private enum class Tab { CHAT, STAGE, PROFILE }

/// Tiny ViewModelProvider.Factory that constructs each ViewModel via a
/// lambda — used to inject AuthRepo's Retrofit/OkHttp into BoxRepo and
/// ChatViewModel without dragging in Hilt for v1.
private class SimpleFactory<T : ViewModel>(
    private val builder: () -> T,
) : ViewModelProvider.Factory {
    @Suppress("UNCHECKED_CAST")
    override fun <V : ViewModel> create(modelClass: Class<V>): V = builder() as V
}
