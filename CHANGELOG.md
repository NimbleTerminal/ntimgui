# Changelog

## 0.3.0 (2026-11-18)

### Breaking change: ntui is now ntimgui

The project is now named **ntimgui** to make its immediate-mode GUI design explicit. The Go module is `nimbleterminal.dev/ntimgui`, and the root package is `ntimgui`.

### Migration

- Replace imports of `nimbleterminal.dev/ntui` with `nimbleterminal.dev/ntimgui`, including the `/ebiten` subpackage.
- Update root package qualifiers from `ntui` to `ntimgui` (or retain an explicit import alias while migrating).
- Native example binaries now use the `ntimgui-` prefix.

Install this release after publication:

```sh
go get nimbleterminal.dev/ntimgui@v0.3.0
```

The Ebitengine backend and standalone example from v0.2.0 are preserved under the renamed module.

### Dependencies and browser demos

- Migrate Booba from `github.com/NimbleMarkets/go-booba` to `nimbleterminal.dev/booba v0.13.0`.
- Update Bubble Tea to v2.1.2, Ebitengine to v2.10.6, ntcanvas and ntpicture to v0.2.0, and terminfo to v1.3.1. Refresh Ultraviolet and other transitive dependencies.
- Update the browser-only Bubble Tea fork to Booba's compatible pinned revision.
- Refresh ntwebterm browser runtime asset checks, documentation, and third-party license notices.

### Validation

- Add visual regression tests for slider thumb placement, track fill, and color editor swatches in the glyph and raster backends. These tests passed in the local native and WebAssembly CI runs.
- Full `task ci` passed on Linux and macOS for commit `6dbbe49a3a5532f12dedecf751d4f676a087f1ce`.
- Native race tests, no-CGO checks, Windows compilation, WASM tests, and all six browser demo smoke tests passed.
- GitHub Pages build and deployment succeeded.

[CI results](https://github.com/NimbleTerminal/ntimgui/actions/runs/42126381524) · [Pages deployment](https://github.com/NimbleTerminal/ntimgui/actions/runs/42126381566)

**Full changelog:** https://github.com/NimbleTerminal/ntimgui/compare/v0.2.0...v0.3.0
