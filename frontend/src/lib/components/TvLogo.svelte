<script lang="ts">
	import { resolveTvChannel } from '$lib/tvChannels';

	let { channel = '', compact = false }: { channel?: string; compact?: boolean } = $props();

	let logo = $derived(resolveTvChannel(channel));
</script>

{#if logo}
	<span
		class="tv-logo {logo.id}"
		class:compact
		class:fullbleed={logo.fullBleed}
		style={`--tv-plate: ${logo.plate}; --tv-border: ${logo.border};`}
		title={logo.label}
		aria-label={logo.label}
	>
		<img src={logo.src} alt={logo.label} loading="lazy" decoding="async" />
	</span>
{:else if channel}
	<span class="tv-logo fallback" class:compact>{channel}</span>
{/if}

<style>
	.tv-logo {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 90px;
		height: 26px;
		padding: 0.14rem 0.36rem;
		border: 1px solid var(--tv-border, var(--border));
		border-radius: 8px;
		background: var(--tv-plate, var(--surface));
		overflow: hidden;
		vertical-align: middle;
		box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.04);
	}
	.tv-logo.compact {
		width: 70px;
		height: 21px;
		padding: 0.1rem 0.26rem;
		border-radius: 7px;
	}
	.tv-logo img {
		display: block;
		max-width: 100%;
		max-height: 100%;
		object-fit: contain;
	}
	.tv-logo.fullbleed {
		padding: 0;
	}
	.tv-logo.fullbleed img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.tv-logo.nrk.fullbleed img {
		width: 88%;
		height: 88%;
		object-fit: contain;
		border-radius: 5px;
	}
	.tv-logo.tv2 {
		box-shadow:
			inset 0 0 0 1px rgba(255, 255, 255, 0.08),
			0 8px 18px -16px rgba(0, 0, 0, 0.65);
	}
	.tv-logo.fallback {
		width: auto;
		min-width: 54px;
		color: var(--muted);
		font-size: 0.72rem;
		font-weight: 850;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}
</style>