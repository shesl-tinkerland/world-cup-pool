<script lang="ts">
	import { pwa } from '$lib/pwa.svelte';
	import { language } from '$lib/language.svelte';
	import { strings } from '$lib/strings';
	import { X, Share } from '@lucide/svelte';
	const t = $derived(strings[language.resolved]);
</script>

{#if pwa.bannerOpen}
	<div class="banner" role="region" aria-label="Installer app">
		<div class="inner">
			<img class="appicon" src="/favicon.svg" alt="" />
			<div class="msg">
				<strong>{t.pwa.installTitle}</strong>
				<span class="muted small">{t.pwa.installBody}</span>
			</div>
			<button class="btn install" onclick={() => pwa.install()}>{t.pwa.installButton}</button>
			<button
				class="x"
				aria-label={t.pwa.close}
				onclick={() => pwa.dismissBanner()}
			>
				<X size={16} />
			</button>
		</div>
	</div>
{/if}

{#if pwa.iosHelpOpen}
	<button
		type="button"
		class="ios-backdrop"
		aria-label="Lukk"
		onclick={() => pwa.closeIosHelp()}
	></button>
	<div class="ios-sheet" role="dialog" aria-label={t.pwa.iosTitle}>
		<h3>{t.pwa.iosTitle}</h3>
		<ol>
			<li>
				{t.pwa.iosStep1.split('Del-knappen')[0]}<span class="kbd"><Share size={14} /> Del</span>{t.pwa.iosStep1.split('Del-knappen')[1] ?? ''}
			</li>
			<li>{t.pwa.iosStep2}</li>
			<li>{t.pwa.iosStep3}</li>
		</ol>
		<button class="btn" onclick={() => pwa.closeIosHelp()}>{t.pwa.understood}</button>
	</div>
{/if}

<style>
	.banner {
		margin: 0 0 1rem;
		padding: 0.7rem 0.85rem;
		background: color-mix(in srgb, var(--accent) 18%, var(--surface));
		border: 1px solid color-mix(in srgb, var(--accent) 35%, var(--border));
		border-radius: var(--radius);
	}
	.inner {
		display: flex;
		align-items: center;
		gap: 0.65rem;
	}
	.appicon {
		width: 38px;
		height: 38px;
		border-radius: 10px;
		flex: none;
	}
	.msg {
		display: flex;
		flex-direction: column;
		min-width: 0;
		flex: 1;
		line-height: 1.25;
	}
	.msg strong {
		font-size: 0.95rem;
	}
	.msg .small {
		font-size: 0.78rem;
	}
	.btn.install {
		padding: 0.45rem 0.85rem;
		font-size: 0.85rem;
		width: auto;
	}
	.x {
		display: inline-grid;
		place-items: center;
		width: 32px;
		height: 32px;
		border-radius: 999px;
		background: transparent;
		color: var(--muted);
		border: 1px solid transparent;
		cursor: pointer;
	}
	.x:hover {
		color: var(--text);
		background: var(--surface-2);
	}

	/* iOS coaching sheet — a small bottom-anchored card with steps. */
	.ios-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.45);
		border: none;
		padding: 0;
		z-index: 60;
		cursor: pointer;
	}
	.ios-sheet {
		position: fixed;
		inset: auto 0.75rem calc(var(--nav-h, 0px) + 0.75rem) 0.75rem;
		z-index: 61;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 1rem 1.1rem 1.1rem;
		box-shadow: var(--shadow-pop);
		max-width: 420px;
		margin: 0 auto;
	}
	.ios-sheet h3 {
		margin: 0 0 0.65rem;
		font-size: 1.05rem;
	}
	.ios-sheet ol {
		margin: 0 0 1rem;
		padding-left: 1.25rem;
		line-height: 1.55;
		font-size: 0.92rem;
	}
	.ios-sheet ol li + li {
		margin-top: 0.4rem;
	}
	.kbd {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		padding: 0.1rem 0.4rem;
		border: 1px solid var(--border);
		border-radius: 4px;
		background: var(--surface-2);
		font-size: 0.85em;
	}

	/* Mobile-only — desktop users don't see either surface. */
	@media (min-width: 900px) {
		.banner,
		.ios-backdrop,
		.ios-sheet {
			display: none;
		}
	}
</style>
