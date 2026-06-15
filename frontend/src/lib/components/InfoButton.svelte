<script lang="ts">
	import { Info } from '@lucide/svelte';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';

	const t = $derived(strings[language.resolved]);
</script>

<a class="info-button" href="/info" aria-label={t.chrome.about} title={t.chrome.about}>
	<Info size={18} aria-hidden="true" />
	<span>Info</span>
</a>

<style>
	.info-button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.4rem;
		width: auto;
		height: 38px;
		padding: 0 0.85rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		color: var(--text);
		font-weight: 600;
		font-size: 0.85rem;
		cursor: pointer;
		transition:
			background 0.15s ease,
			border-color 0.15s ease,
			color 0.15s ease,
			transform 0.15s ease;
	}
	.info-button:hover {
		border-color: var(--border-strong);
		background: var(--surface-3);
		color: var(--accent);
	}
	.info-button:active {
		transform: scale(0.96);
	}
	.info-button:focus-visible {
		outline: var(--ring);
		outline-offset: 2px;
	}
</style>