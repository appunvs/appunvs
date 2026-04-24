# Icon placeholders

Tauri requires real icon assets (`32x32.png`, `128x128.png`, `128x128@2x.png`,
`icon.icns`, `icon.ico`) to produce bundles. None are committed here.

Generate them from a source PNG after the first `cargo check` / `npm install`:

```
npx @tauri-apps/cli icon path/to/source.png
```

This populates `src-tauri/icons/` with all required formats.
