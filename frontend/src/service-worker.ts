/// <reference types="@sveltejs/kit" />
/// <reference lib="webworker" />

// Minimal offline-capable service worker (makes the app installable and the
// shell available offline). API/auth calls always go to the network.
import { build, files, version } from '$service-worker';

const sw = self as unknown as ServiceWorkerGlobalScope;
const CACHE = `wmtips-${version}`;

// Precache the app shell + light static assets. Skip the heavy stuff
// (flags / screenshots) — those are cached on demand instead.
const PRECACHE = [
	...build,
	...files.filter(
		(f) => !f.startsWith('/flags/') && !f.startsWith('/screenshots/')
	)
];

sw.addEventListener('install', (e) => {
	e.waitUntil(
		caches
			.open(CACHE)
			.then(async (c) => {
				await c.addAll(PRECACHE);
				// App-shell entry (adapter-static fallback) — best effort.
				await c.add('/').catch(() => {});
			})
			.then(() => sw.skipWaiting())
	);
});

sw.addEventListener('activate', (e) => {
	e.waitUntil(
		caches
			.keys()
			.then((keys) =>
				Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))
			)
			.then(() => sw.clients.claim())
	);
});

sw.addEventListener('fetch', (e) => {
	const req = e.request;
	const url = new URL(req.url);

	// Only handle same-origin GETs; never the API / PocketBase routes.
	if (
		req.method !== 'GET' ||
		url.origin !== location.origin ||
		url.pathname.startsWith('/api/') ||
		url.pathname.startsWith('/_/')
	) {
		return;
	}

	// SPA navigations: serve the cached app shell when offline.
	if (req.mode === 'navigate') {
		e.respondWith(
			fetch(req).catch(
				async () =>
					(await caches.match('/')) ??
					(await caches.match('/index.html')) ??
					Response.error()
			)
		);
		return;
	}

	// Static assets: cache-first, fall back to network and cache the result.
	e.respondWith(
		caches.match(req).then(
			(hit) =>
				hit ??
				fetch(req).then((res) => {
					if (res.ok && res.type === 'basic') {
						const copy = res.clone();
						caches.open(CACHE).then((c) => c.put(req, copy));
					}
					return res;
				})
		)
	);
});

sw.addEventListener('push', (e) => {
	const event = e as ExtendableEvent & { data?: PushMessageData | null };
	let data: Record<string, string> = {};
	try {
		data = event.data ? event.data.json() : {};
	} catch {
		data = { title: 'VM Tipping', body: event.data ? event.data.text() : '' };
	}

	const title = data.title || 'VM Tipping';
	const options: NotificationOptions = {
		body: data.body || '',
		icon: '/icons/maskable_icon_x192.png',
		badge: '/icons/maskable_icon_x192.png',
		tag: data.tag || 'vm-tipping',
		renotify: true,
		data: { url: data.url || '/' }
	};

	event.waitUntil(sw.registration.showNotification(title, options));
});

sw.addEventListener('notificationclick', (e) => {
	const event = e as ExtendableEvent & { notification: Notification };
	event.notification.close();
	const target =
		(event.notification.data as { url?: string } | undefined)?.url || '/';

	event.waitUntil(
		sw.clients
			.matchAll({ type: 'window', includeUncontrolled: true })
			.then((clientList) => {
				for (const client of clientList) {
					if ('navigate' in client) {
						void client.navigate(target).catch(() => {});
						return client.focus();
					}
				}
				return sw.clients.openWindow?.(target);
			})
	);
});
