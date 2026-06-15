<script lang="ts">
	// Renders the bundled /flags/<iso2>.svg; falls back to the FIFA code chip
	// when the ISO code is missing (e.g. unresolved knockout slot).
	let {
		iso2 = '',
		code = '',
		size = 22
	}: { iso2?: string; code?: string; size?: number } = $props();
	const visualSizeBoost = 2;
	let failed = $state(false);
</script>

{#if iso2 && !failed}
	<img
		class="flag"
		src={`/flags/${iso2}.svg`}
		alt={code}
		style="width:{size + visualSizeBoost}px;height:{(size + visualSizeBoost) * 0.72}px"
		onerror={() => (failed = true)}
		loading="lazy"
	/>
{:else}
	<span class="flag chip" style="font-size:{(size + visualSizeBoost) * 0.42}px">{code || '—'}</span>
{/if}

<style>
	.flag {
		display: inline-block;
		object-fit: cover;
		border-radius: 3px;
		border: 1px solid var(--border);
		flex: none;
		vertical-align: middle;
	}
	.chip {
		display: inline-grid;
		place-items: center;
		min-width: 2.2em;
		padding: 0.15em 0.35em;
		background: var(--surface-2);
		color: var(--muted);
		font-weight: 700;
		letter-spacing: 0.03em;
	}
</style>
