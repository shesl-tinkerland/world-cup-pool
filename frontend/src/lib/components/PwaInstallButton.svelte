<script lang="ts">
	import { pwa } from '$lib/pwa.svelte';
</script>

{#if pwa.available}
	<button
		type="button"
		class="btn install"
		class:outlined={!pwa.dismissed}
		aria-label="Installer VM Tipping"
		onclick={() => pwa.install()}
	>
		Installer
	</button>
{/if}

<style>
	.install {
		/* Override .btn's default full-width broadcast styling so this fits
		   inline in the topbar next to the avatar. Keep the default
		   --radius-sm corners from .btn. */
		width: auto;
		padding: 0.45rem 0.85rem;
		font-size: 0.78rem;
		flex: none;
	}
	/* Outlined while the first-visit banner is still around to do the
	   talking — once the banner is dismissed, the topbar button takes
	   over the CTA role and goes back to the filled accent style. */
	.install.outlined {
		background-color: transparent;
		border-color: var(--accent);
		color: var(--accent);
	}
	@media (min-width: 900px) {
		/* Mobile-only — we only push installs on phones. */
		.install {
			display: none;
		}
	}
</style>
