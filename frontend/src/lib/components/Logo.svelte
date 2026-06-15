<script lang="ts">
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';

	let {
		compact = false,
		variant = 'default',
		tagline = ''
	}: { compact?: boolean; variant?: 'default' | 'hero'; tagline?: string } =
		$props();
	const t = $derived(strings[language.resolved]);
</script>

<a class="logo" class:compact class:hero={variant === 'hero'} href="/" aria-label={`VM Tipping — ${t.nav.home.toLowerCase()}`}>
	<span class="mark-frame"><img class="mark" src="/logos/fhun_logo-icon.svg" alt="" /></span>
	{#if !compact}
		<span class="copy">
			<span class="word">VM&nbsp;Tipping</span>
			{#if variant === 'hero' && tagline}<span class="tagline">{tagline}</span>{/if}
		</span>
	{/if}
</a>

<style>
	.logo {
		display: inline-flex;
		align-items: center;
		gap: 0.8rem;
		color: var(--text);
		font-weight: 800;
		letter-spacing: 0;
	}
	.mark-frame {
		display: inline-grid;
		place-items: center;
		width: 40px;
		height: 40px;
		border-radius: 14px;
		background: color-mix(in srgb, var(--gold) 12%, var(--surface-2));
		box-shadow: var(--shadow-tile);
		flex: none;
	}
	.mark {
		height: 34px;
		width: auto;
		display: block;
	}
	.copy {
		display: grid;
		gap: 0.12rem;
	}
	.word {
		font-family: var(--font-display);
		font-size: 1.35rem;
		font-weight: 700;
		letter-spacing: 0;
		line-height: 1;
	}
	.tagline {
		font-size: 0.82rem;
		font-weight: 650;
		color: var(--muted);
		line-height: 1.25;
	}
	.logo.hero {
		align-items: center;
		gap: 1rem;
	}
	.logo.hero .mark-frame {
		width: 68px;
		height: 68px;
		border-radius: 20px;
		background:
			linear-gradient(145deg, color-mix(in srgb, var(--gold) 24%, var(--surface)), var(--surface-2));
		border: 1px solid color-mix(in srgb, var(--gold) 30%, var(--border));
		box-shadow: var(--shadow-pop);
	}
	.logo.hero .mark {
		height: 58px;
	}
	.logo.hero .word {
		font-size: clamp(2rem, 9vw, 3rem);
		font-weight: 800;
	}
	.logo.hero .tagline {
		font-size: 0.95rem;
	}
	.logo.compact {
		gap: 0;
	}
	.logo.compact .mark-frame {
		width: 38px;
		height: 38px;
		border-radius: 13px;
	}
	.logo.compact .mark {
		height: 32px;
	}
</style>
