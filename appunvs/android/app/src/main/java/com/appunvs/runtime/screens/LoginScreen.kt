// LoginScreen — email + password gate.  Mirrors iOS LoginView.  One
// form, toggleable between Sign in / Sign up via a segmented control.
package com.appunvs.runtime.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.state.AuthRepo
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Spacing
import com.appunvs.runtime.ui.AppCard

private enum class LoginMode { LOGIN, SIGNUP }

@Composable
fun LoginScreen(
    auth: AuthRepo,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current
    var mode by remember { mutableStateOf(LoginMode.LOGIN) }
    var email by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    val canSubmit = email.contains('@') && password.length >= 6

    Box(
        modifier = modifier
            .fillMaxSize()
            .background(colors.bgPage)
            .padding(horizontal = Spacing.l.dp),
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = Spacing.huge.dp),
            verticalArrangement = Arrangement.spacedBy(Spacing.l.dp),
        ) {
            Text(
                text = "appunvs",
                style = MaterialTheme.typography.displaySmall.copy(
                    color = colors.textPrimary,
                    fontWeight = FontWeight.Bold,
                ),
            )
            Text(
                text = "聊一句, 跑一个 app",
                style = MaterialTheme.typography.bodyMedium.copy(color = colors.textSecondary),
            )

            AppCard(modifier = Modifier.fillMaxWidth()) {
                Column(verticalArrangement = Arrangement.spacedBy(Spacing.l.dp)) {
                    val tabs = listOf(LoginMode.LOGIN to "登录", LoginMode.SIGNUP to "注册")
                    SingleChoiceSegmentedButtonRow(modifier = Modifier.fillMaxWidth()) {
                        tabs.forEachIndexed { index, (value, label) ->
                            SegmentedButton(
                                selected = mode == value,
                                onClick = { mode = value },
                                shape = SegmentedButtonDefaults.itemShape(index = index, count = tabs.size),
                            ) { Text(label) }
                        }
                    }

                    OutlinedTextField(
                        value = email,
                        onValueChange = { email = it },
                        label = { Text("邮箱") },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
                        modifier = Modifier.fillMaxWidth(),
                    )

                    OutlinedTextField(
                        value = password,
                        onValueChange = { password = it },
                        label = { Text("密码") },
                        singleLine = true,
                        visualTransformation = PasswordVisualTransformation(),
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                        modifier = Modifier.fillMaxWidth(),
                    )

                    auth.lastError?.let { err ->
                        Text(
                            text = err,
                            style = MaterialTheme.typography.bodySmall.copy(color = colors.semanticDanger),
                        )
                    }

                    val isBootstrapping = auth.phase == AuthRepo.Phase.Bootstrapping
                    Button(
                        onClick = {
                            val e = email.trim()
                            when (mode) {
                                LoginMode.LOGIN  -> auth.login(e, password)
                                LoginMode.SIGNUP -> auth.signup(e, password)
                            }
                        },
                        enabled = canSubmit && !isBootstrapping,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            if (isBootstrapping) {
                                CircularProgressIndicator(
                                    modifier = Modifier.width(16.dp),
                                    strokeWidth = 2.dp,
                                    color = colors.bgCard,
                                )
                                Spacer(Modifier.width(Spacing.s.dp))
                            }
                            Text(if (mode == LoginMode.LOGIN) "登录" else "创建账号")
                        }
                    }
                }
            }

            Text(
                text = "登录后将自动注册当前设备。",
                style = MaterialTheme.typography.bodySmall.copy(color = colors.textSecondary),
            )
        }
    }
}
