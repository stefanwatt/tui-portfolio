<script lang="ts">
	import { mocha } from '$lib/catppuccin';

	let term: any;
	let lineBuffer = '';

	function terminal(node: HTMLElement) {
		//@ts-ignore
		term = new window.Terminal({ theme: mocha });
		term.open(node);
		term.write('Hello from \x1B[1;3;31mStefan\x1B[0m\r\n$ '); // newline before prompt
		setupWasm();
	}

	async function setupWasm() {
		//@ts-ignore
		const go = new Go();
		const wasmResp = await fetch('/main.wasm');
		const wasmBuffer = await wasmResp.arrayBuffer();
		const { instance } = await WebAssembly.instantiate(wasmBuffer, go.importObject);
		go.run(instance);

		(window as any).receiveLine = (line: string) => {
			term.write('\r\n' + line + '\r\n$ '); // newline before and after Go output
		};

		term.onData((data: string) => {
			for (let ch of data) {
				if (ch === '\r') {
					(window as any).sendLine(lineBuffer);
					lineBuffer = '';
				} else if (ch === '\u007F') {
					if (lineBuffer.length > 0) {
						lineBuffer = lineBuffer.slice(0, -1);
						term.write('\b \b');
					}
				} else {
					lineBuffer += ch;
					term.write(ch);
				}
			}
		});
	}
</script>

<div use:terminal></div>
