// MainActivity — single Activity hosting the entire Compose tree.
// Three top-level tabs (Chat / Stage / Profile) live inside; nothing
// else routes to its own Activity in this design.
//
// Future: when SubRuntime native module lands, the Stage tab embeds
// a native View that hosts a sandboxed Hermes runtime; everything
// else stays Compose.
package com.appunvs.runtime

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AccountCircle
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.PlayCircleOutline
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.viewmodel.compose.viewModel

import com.appunvs.runtime.screens.ChatScreen
import com.appunvs.runtime.screens.ProfileScreen
import com.appunvs.runtime.screens.StageScreen
import com.appunvs.runtime.state.AppState
import com.appunvs.runtime.theme.RuntimeTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            RuntimeRoot()
        }
    }
}

@Composable
fun RuntimeRoot(state: AppState = viewModel()) {
    RuntimeTheme(themeOverride = state.themeOverride) {
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
            when (selected) {
                Tab.CHAT    -> ChatScreen(modifier = Modifier.fillMaxSize().padding(padding))
                Tab.STAGE   -> StageScreen(modifier = Modifier.fillMaxSize().padding(padding))
                Tab.PROFILE -> ProfileScreen(modifier = Modifier.fillMaxSize().padding(padding), state = state)
            }
        }
    }
}

private enum class Tab { CHAT, STAGE, PROFILE }
