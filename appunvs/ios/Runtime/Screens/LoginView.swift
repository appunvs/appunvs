// LoginView — email + password gate.  One form, toggleable between
// Sign in / Sign up via a segmented control.  Other auth methods
// (SMS / WeChat / Apple) are deferred — the relay only supports
// email/password today.
import SwiftUI

struct LoginView: View {
    @EnvironmentObject private var auth: AuthStore

    @State private var mode: Mode = .login
    @State private var email: String = ""
    @State private var password: String = ""
    @State private var submitting: Bool = false

    enum Mode: Hashable { case login, signup }

    var body: some View {
        ZStack {
            Theme.bgPage.color.ignoresSafeArea()
            VStack(spacing: Spacing.l) {
                header
                Card {
                    VStack(alignment: .leading, spacing: Spacing.l) {
                        Picker("", selection: $mode) {
                            Text("登录").tag(Mode.login)
                            Text("注册").tag(Mode.signup)
                        }
                        .pickerStyle(.segmented)

                        field(label: "邮箱", text: $email)
                            .keyboardType(.emailAddress)
                            .textContentType(.emailAddress)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()

                        secureField(label: "密码", text: $password)

                        if let err = auth.lastError {
                            Text(err)
                                .font(.caption)
                                .foregroundStyle(Theme.semanticDanger.color)
                        }

                        Button(action: submit) {
                            HStack {
                                if submitting { ProgressView().tint(.white) }
                                Text(mode == .login ? "登录" : "创建账号")
                                    .font(.body.weight(.semibold))
                                    .foregroundStyle(.white)
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, Spacing.m)
                            .background(
                                RoundedRectangle(cornerRadius: Radius.m)
                                    .fill(canSubmit ? Theme.brandDark.color : Theme.bgInput.color)
                            )
                        }
                        .buttonStyle(.plain)
                        .disabled(!canSubmit || submitting)
                    }
                }

                Text("登录后将自动注册当前设备。")
                    .font(.footnote)
                    .foregroundStyle(Theme.textSecondary.color)
                    .multilineTextAlignment(.center)

                Spacer()
            }
            .padding(.horizontal, Spacing.l)
            .padding(.top, Spacing.xxl)
        }
    }

    private var header: some View {
        VStack(spacing: Spacing.s) {
            Text("appunvs")
                .font(.largeTitle.weight(.bold))
                .foregroundStyle(Theme.textPrimary.color)
            Text("聊一句, 跑一个 app")
                .font(.subheadline)
                .foregroundStyle(Theme.textSecondary.color)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var canSubmit: Bool {
        let e = email.trimmingCharacters(in: .whitespaces)
        return !e.isEmpty && password.count >= 6 && e.contains("@")
    }

    private func field(label: String, text: Binding<String>) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(Theme.textSecondary.color)
            TextField("", text: text)
                .padding(Spacing.m)
                .background(
                    RoundedRectangle(cornerRadius: Radius.m)
                        .fill(Theme.bgInput.color)
                )
        }
    }

    private func secureField(label: String, text: Binding<String>) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(Theme.textSecondary.color)
            SecureField("", text: text)
                .padding(Spacing.m)
                .background(
                    RoundedRectangle(cornerRadius: Radius.m)
                        .fill(Theme.bgInput.color)
                )
        }
    }

    private func submit() {
        let e = email.trimmingCharacters(in: .whitespaces)
        submitting = true
        Task {
            switch mode {
            case .login:  await auth.login(email: e, password: password)
            case .signup: await auth.signup(email: e, password: password)
            }
            submitting = false
        }
    }
}

#Preview {
    LoginView()
        .environmentObject(AuthStore())
}
