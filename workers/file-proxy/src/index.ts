interface Env {
	NOTES_BUCKET: R2Bucket;
	FILE_PROXY_SECRET: string;
}

type SignedPayload = {
	key: string;
	exp: number;
};

const encoder = new TextEncoder();

export default {
	async fetch(request: Request, env: Env): Promise<Response> {
		if (request.method !== 'GET' && request.method !== 'HEAD') {
			return new Response('Method Not Allowed', {
				status: 405,
				headers: {
					Allow: 'GET, HEAD',
				},
			});
		}

		const url = new URL(request.url);

		if (url.pathname !== '/file') {
			return new Response('Not Found', { status: 404 });
		}

		const token = url.searchParams.get('token');
		const signature = url.searchParams.get('signature');

		if (!token || !signature) {
			return new Response('Missing file token', { status: 400 });
		}

		const payload = await verifySignedToken(token, signature, env.FILE_PROXY_SECRET);

		if (!payload) {
			return new Response('Invalid file token', { status: 403 });
		}

		const now = Math.floor(Date.now() / 1000);
		if (payload.exp < now) {
			return new Response('File token expired', { status: 403 });
		}

		const object = await env.NOTES_BUCKET.get(payload.key);

		if (!object) {
			return new Response('File not found', { status: 404 });
		}

		const headers = new Headers();

		object.writeHttpMetadata(headers);

		headers.set('etag', object.httpEtag);
		headers.set('cache-control', 'private, max-age=300');
		headers.set('x-content-type-options', 'nosniff');
		headers.set('content-security-policy', "default-src 'none'; frame-ancestors 'self'; sandbox");
		headers.set('referrer-policy', 'no-referrer');

		const fileName = safeFileName(object.key);

		if (!headers.has('content-type')) {
			headers.set('content-type', 'application/octet-stream');
		}

		headers.set('content-disposition', `inline; filename="${fileName}"`);

		if (request.method === 'HEAD') {
			return new Response(null, {
				status: 200,
				headers,
			});
		}

		return new Response(object.body, {
			status: 200,
			headers,
		});
	},
};

async function verifySignedToken(token: string, signature: string, secret: string): Promise<SignedPayload | null> {
	try {
		const expectedSignature = await hmacSHA256Base64URL(secret, token);

		if (!constantTimeEqual(signature, expectedSignature)) {
			return null;
		}

		const decoded = base64URLDecode(token);
		const payload = JSON.parse(decoded) as SignedPayload;

		if (!payload.key || typeof payload.key !== 'string') {
			return null;
		}

		if (!Number.isInteger(payload.exp)) {
			return null;
		}

		if (payload.key.startsWith('/') || payload.key.includes('..')) {
			return null;
		}

		return payload;
	} catch {
		return null;
	}
}

async function hmacSHA256Base64URL(secret: string, value: string): Promise<string> {
	const key = await crypto.subtle.importKey(
		'raw',
		encoder.encode(secret),
		{
			name: 'HMAC',
			hash: 'SHA-256',
		},
		false,
		['sign'],
	);

	const signature = await crypto.subtle.sign('HMAC', key, encoder.encode(value));

	return arrayBufferToBase64URL(signature);
}

function constantTimeEqual(a: string, b: string): boolean {
	const aBytes = encoder.encode(a);
	const bBytes = encoder.encode(b);

	if (aBytes.length !== bBytes.length) {
		return false;
	}

	let result = 0;

	for (let i = 0; i < aBytes.length; i++) {
		result |= aBytes[i] ^ bBytes[i];
	}

	return result === 0;
}

function base64URLDecode(value: string): string {
	const base64 = value.replace(/-/g, '+').replace(/_/g, '/');
	const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=');

	const binary = atob(padded);
	const bytes = new Uint8Array(binary.length);

	for (let i = 0; i < binary.length; i++) {
		bytes[i] = binary.charCodeAt(i);
	}

	return new TextDecoder().decode(bytes);
}

function arrayBufferToBase64URL(buffer: ArrayBuffer): string {
	const bytes = new Uint8Array(buffer);
	let binary = '';

	for (const byte of bytes) {
		binary += String.fromCharCode(byte);
	}

	return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
}

function safeFileName(key: string): string {
	const parts = key.split('/');
	const last = parts[parts.length - 1] || 'file';

	return last.replace(/["\\\r\n]/g, '_');
}
