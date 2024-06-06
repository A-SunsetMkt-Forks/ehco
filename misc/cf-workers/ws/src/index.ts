/**
 * Welcome to Cloudflare Workers! This is your first worker.
 *
 * - Run `npm run dev` in your terminal to start a development server
 * - Open a browser tab at http://localhost:8787/ to see your worker in action
 * - Run `npm run deploy` to publish your worker
 *
 * Bind resources to your worker in `wrangler.toml`. After adding bindings, a type definition for the
 * `Env` object can be regenerated with `npm run cf-typegen`.
 *
 * Learn more at https://developers.cloudflare.com/workers/
 */

async function handleRequest(request) {
	const upgradeHeader = request.headers.get('Upgrade');
	if (!upgradeHeader || upgradeHeader !== 'websocket') {
		return new Response('Expected Upgrade: websocket', { status: 426 });
	}

	const webSocketPair = new WebSocketPair();
	const [client, server] = Object.values(webSocketPair);

	server.accept();

	// Connect to the upstream WebSocket server
	const upstreamWebSocket = new WebSocket('ws://0.0.0.0:2443/pwd');

	upstreamWebSocket.onopen = () => {
		console.log('Connected to upstream WebSocket server');
	};

	upstreamWebSocket.onmessage = (event) => {
		console.log('Message from upstream:', event.data);
		server.send(event.data);
	};

	upstreamWebSocket.onclose = () => {
		console.log('Upstream WebSocket closed');
		server.close();
	};

	upstreamWebSocket.onerror = (error) => {
		console.error('Upstream WebSocket error:', error);
		server.close();
	};

	server.addEventListener('message', (msg) => {
		console.log('Message from client:', msg.data);
		upstreamWebSocket.send(msg.data);
	});

	server.addEventListener('close', () => {
		console.log('Client WebSocket closed');
		upstreamWebSocket.close();
	});

	return new Response(null, {
		status: 101,
		webSocket: client,
	});
}

addEventListener('fetch', (event) => {
	event.respondWith(handleRequest(event.request));
});

export default {
	async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
		return handleRequest(request)
	},
};
