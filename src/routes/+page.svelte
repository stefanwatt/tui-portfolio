<script lang="ts">
	import { mocha } from '$lib/catppuccin';
	let term: any;

	function setupTerminal(node: HTMLElement) {
		(async () => {
			// Load Go WASM runtime
			// @ts-ignore
			const go = new Go();
			const wasmResp = await fetch('/main.wasm');
			const wasmBuffer = await wasmResp.arrayBuffer();
			const { instance } = await WebAssembly.instantiate(wasmBuffer, go.importObject);
			go.run(instance);
			//@ts-ignore
			if (!window.bubbletea_write) {
				console.log('bubbletea not ready....gotta wait');
				setTimeout(() => setupTerminal(node), 500);
				return;
			}
			// @ts-ignore
			term = new window.Terminal({ theme: mocha });
			term.open(node);

			// @ts-ignore
			const fitAddon = new window.FitAddon.FitAddon();
			term.loadAddon(fitAddon);

			term.focus();
			// @ts-ignore
			window.bubbletea_resize(term.cols, term.rows);

			setInterval(() => {
				// @ts-ignore
				const out = window.bubbletea_read();
				if (out) term.write(out);
			}, 50);

			// @ts-ignore
			term.onData((d) => window.bubbletea_write(d));
			// @ts-ignore
			term.onResize(() => window.bubbletea_resize(term.cols, term.rows));
			window.addEventListener('resize', () => fitAddon.fit());
		})();
	}
</script>

<div use:setupTerminal class="h-full w-full"></div>
