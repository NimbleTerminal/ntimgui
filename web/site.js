import { BoobaTerminal, installActionBar, installRendererHud, parseRendererFromURL } from './booba/booba.js';

const demos = {
  catalog: 'Controls & input',
  dashboard: 'Telemetry dashboard',
  hud: '3D viewport + HUD',
};
const catalogTabs = {
  plots: { flag: '1', title: 'Plots & sparklines' },
  trees: { flag: '2', title: 'Trees & lists' },
  themes: { flag: '3', title: 'Themes & palettes' },
};
const help = {
  catalog: 'Click tabs, drag panel title bars, or use ] to switch tabs · t: cycle theme · b: cycle border · g: toggle Glyph/Raster · Tab: focus next widget · q: quit.',
  dashboard: 'Space: pause/resume telemetry · t: cycle theme · g: toggle Glyph/Raster · Tab: focus next widget · q: quit.',
  hud: 'Drag panels or adjust sliders · drag background to orbit 3D camera · mouse wheel or +/-: zoom · Space: pause/orbit · m: cycle mesh · r: reset camera · g: toggle Glyph/Raster · q: quit.',
};
const params = new URLSearchParams(location.search);
const requested = params.get('demo');
const demo = Object.hasOwn(demos, requested) ? requested : 'catalog';
const tabKey = demo === 'catalog' ? params.get('tab') : null;
const tabPreset = tabKey && Object.hasOwn(catalogTabs, tabKey) ? catalogTabs[tabKey] : null;
const navKey = tabPreset ? tabKey : demo;
const title = tabPreset ? tabPreset.title : demos[demo];

const status = document.querySelector('#status');
document.querySelector('#title').textContent = title;
document.title = `${title} · ntimgui`;
const restart = new URL(location.href);
restart.searchParams.set('demo', demo);
if (tabPreset) {
  restart.searchParams.set('tab', tabKey);
} else {
  restart.searchParams.delete('tab');
}
document.querySelector('#restart').href = restart.href;
document.querySelector('#source').href = `https://github.com/NimbleTerminal/ntimgui/tree/main/examples/${demo}`;
for (const link of document.querySelectorAll('[data-demo]')) {
  if (link.dataset.demo === navKey) link.setAttribute('aria-current', 'page');
}
document.querySelector('#help').textContent = help[demo];
function failed(error) {
  status.dataset.error = '';
  status.textContent = `Could not run the demo: ${error.message}. Try Restart.`;
  console.error(error);
}
async function main() {
  const go = new Go();
  if (tabPreset) {
    go.argv = ['catalog', `-tab=${tabPreset.flag}`];
  }
  const response = await fetch(`demos/${demo}.wasm`);
  if (!response.ok) throw new Error(`Download failed (${response.status})`);
  const { instance } = await WebAssembly.instantiateStreaming(response, go.importObject);
  const running = go.run(instance);
  running.then(() => { status.textContent = 'Session ended. Choose Restart to run again.'; }, failed);
  const terminal = new BoobaTerminal('terminal', { renderer: parseRendererFromURL(), allowOSC7501Actions: true });
  await terminal.init();
  terminal.attachCustomWheelEventHandler(event => {
    if (terminal.hasMouseTracking() && terminal.term?.inputHandler) {
      terminal.term.inputHandler.handleWheel(event);
      return true;
    }
    return false;
  });
  terminal.setActionKeyEncoder({
    canEncode: gesture => terminal.encodeKey(gesture) !== null,
    encode: gesture => terminal.encodeKey(gesture),
  });
  installActionBar(terminal.actions, {
    parent: document.querySelector('#controls'),
    openKeyboard: () => terminal.focus(),
    onLayout: () => terminal.fitAddon?.fit(),
  });
  terminal.onActionError = failed;
  window.addEventListener('pagehide', () => terminal.dispose(), { once: true });
  terminal.connectWasm();
  terminal.focus();
  installRendererHud(terminal.term, {
    parent: document.querySelector('.terminal-frame'),
    position: 'absolute',
    bindToggleHotkey: false,
  });
  status.textContent = 'Running locally in your browser. No terminal server required.';
}
main().catch(failed);
