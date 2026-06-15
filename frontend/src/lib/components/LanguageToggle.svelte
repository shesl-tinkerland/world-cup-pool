<script lang="ts">
	import { Languages } from '@lucide/svelte';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';

	let { compact = false }: { compact?: boolean } = $props();
	const t = $derived(strings[language.resolved]);
</script>

<button
	class="language-toggle"
	class:compact
	type="button"
	aria-label={t.chrome.languageAria}
	title={t.chrome.language}
	onclick={() => language.toggle()}
>
	<Languages size={18} aria-hidden="true" />
	{#if !compact}
		<span>{t.chrome.language}</span>
	{/if}
</button>

<style>
	.language-toggle {
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
	.language-toggle.compact {
		width: 38px;
		padding: 0;
	}
	.language-toggle:hover {
		border-color: var(--border-strong);
		background: var(--surface-3);
		color: var(--accent);
	}
	.language-toggle:active {
		transform: scale(0.96);
	}
	.language-toggle:focus-visible {
		outline: var(--ring);
		outline-offset: 2px;
	}
</style>