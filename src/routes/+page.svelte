<script lang="ts">
	import { mocha } from '$lib/catppuccin';
	let term: any;

	function setupTerminal(node: HTMLElement) {
		(async () => {
			term = new (window as any).Terminal({ theme: mocha, convertEol: true });
			term.open(node);

			const fitAddon = new (window as any).FitAddon.FitAddon();
			term.loadAddon(fitAddon);
			term.focus();
			fitAddon.fit();

			const ws = new WebSocket((location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws');

			ws.addEventListener('open', () => {
				// send initial size
				ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
			});

			ws.addEventListener('message', (ev) => {
				if (typeof ev.data === 'string') {
					// Check if it's a console log message
					try {
						const data = JSON.parse(ev.data);
						if (data.type === 'console') {
							// Log to browser console
							const logMessage = `[Go ${data.level.toUpperCase()}] ${data.message}`;
							switch (data.level) {
								case 'error':
									console.error(logMessage);
									break;
								case 'warn':
									console.warn(logMessage);
									break;
								case 'debug':
									console.debug(logMessage);
									break;
								default:
									console.log(logMessage);
							}
							return; // Don't write to terminal
						}
					} catch (e) {
						// Not JSON, treat as regular terminal output
					}
					
					term.write(ev.data);
				} else {
					// assume text for simplicity; browsers may pass Blob
					(ev.data as Blob).text().then((t) => term.write(t));
				}
			});

			term.onData((d: string) => ws.send(d));
			term.onResize(() => ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows })));
			window.addEventListener('resize', () => fitAddon.fit());
		})();
	}
</script>

<div use:setupTerminal class="h-full w-full"></div>
