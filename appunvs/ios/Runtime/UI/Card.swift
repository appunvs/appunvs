// Card — surface wrapper around themed content.  Use this instead of
// `RoundedRectangle().fill()` ad-hoc so corner radius / padding /
// border style live in one place and inherit theme tokens.
import SwiftUI

struct Card<Content: View>: View {
    let padding: CGFloat
    let cornerRadius: CGFloat
    let bordered: Bool
    let content: () -> Content

    init(
        padding: CGFloat = Spacing.l,
        cornerRadius: CGFloat = Radius.l,
        bordered: Bool = false,
        @ViewBuilder content: @escaping () -> Content
    ) {
        self.padding = padding
        self.cornerRadius = cornerRadius
        self.bordered = bordered
        self.content = content
    }

    var body: some View {
        content()
            .padding(padding)
            .background(
                RoundedRectangle(cornerRadius: cornerRadius)
                    .fill(Theme.bgCard.color)
            )
            .overlay(
                bordered
                    ? RoundedRectangle(cornerRadius: cornerRadius)
                        .stroke(Theme.borderDefault.color, lineWidth: 1)
                    : nil
            )
    }
}

#Preview {
    Card {
        Text("Card body")
    }
    .padding()
}
