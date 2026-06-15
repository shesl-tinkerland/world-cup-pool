<script lang="ts">
	import { Sun, Moon } from '@lucide/svelte';
	import { theme } from '$lib/theme.svelte';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';

	let { compact = false }: { compact?: boolean } = $props();
	const isDark = $derived(theme.resolved === 'dark');
	const t = $derived(strings[language.resolved]);
</script>

<button
	class="theme-toggle"
	class:compact
	type="button"
	aria-label={isDark ? t.chrome.lightTheme : t.chrome.darkTheme}
	title={isDark ? t.chrome.lightTheme : t.chrome.darkTheme}
	onclick={() => theme.toggle()}
>
	{#if isDark}
		<Sun size={compact ? 18 : 18} aria-hidden="true" />
	{:else}
		<Moon size={compact ? 18 : 18} aria-hidden="true" />
	{/if}
</button>

<style>
	.theme-toggle {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 38px;
		height: 38px;
		padding: 0;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		color: var(--text);
		cursor: pointer;
		transition:
			background 0.15s ease,
			border-color 0.15s ease,
			color 0.15s ease,
			transform 0.15s ease;
	}
	.theme-toggle:hover {
		border-color: var(--border-strong);
		background: var(--surface-3);
		color: var(--accent);
	}
	.theme-toggle:active {
		transform: scale(0.96);
	}
	.theme-toggle:focus-visible {
		outline: var(--ring);
		outline-offset: 2px;
	}
</style>
