<script lang="ts">
	import { language } from '$lib/language.svelte';
	import { tipsStore, type Match } from '$lib/tips.svelte';
	import { groupTable, type StandRow } from '$lib/standings';
	import Flag from './Flag.svelte';
	import { ChevronDown } from '@lucide/svelte';

	let {
		matches,
		bestThirds
	}: {
		matches: Match[];
		bestThirds: Set<string>;
	} = $props();

	let open = $state(false);
	let rows = $derived(groupTable(matches, tipsStore.tips));
	let counted = $derived(rows.reduce((sum, row) => sum + row.pld, 0));
	const gd = (row: StandRow) => `${row.gf - row.ga >= 0 ? '+' : ''}${row.gf - row.ga}`;
	const advances = (row: StandRow, index: number) =>
		index < 2 || (index === 2 && bestThirds.has(row.id));
</script>

<div class="gs">
	<button class="gs-toggle" onclick={() => (open = !open)} aria-expanded={open}>
		<span>{language.text('Projisert tabell', 'Projisert tabell', 'Projected table')}</span>
		<ChevronDown size={15} class="gs-cv {open ? 'up' : ''}" />
	</button>

	{#if open}
		{#if counted === 0}
			<p class="muted small note">
				{language.text(
					'Tipp kampane i denne gruppa for å sjå den projiserte tabellen.',
					'Tipp kampane i denne gruppa for å sjå den projiserte tabellen.',
					"Tip this group's matches to see the projected table."
				)}
			</p>
		{:else}
			<table class="gs-tbl">
				<thead>
					<tr>
						<th></th>
						<th class="tl">{language.text('Lag', 'Lag', 'Team')}</th>
						<th>P</th>
						<th>GD</th>
						<th>{language.text('Po', 'Po', 'Pts')}</th>
					</tr>
				</thead>
				<tbody>
					{#each rows as row, index (row.id)}
						<tr class:adv={advances(row, index)} class:third={index === 2 && bestThirds.has(row.id)}>
							<td class="rk">{index + 1}</td>
							<td class="tl">
								<span class="tm">
									<Flag
										iso2={tipsStore.team(row.id)?.iso2 ?? ''}
										code={tipsStore.team(row.id)?.fifaCode ?? ''}
									/>
									<span class="nm">{tipsStore.team(row.id)?.name ?? row.id}</span>
								</span>
							</td>
							<td>{row.pld}</td>
							<td>{gd(row)}</td>
							<td class="pts">{row.pts}</td>
						</tr>
					{/each}
				</tbody>
			</table>
			<p class="muted small note">
				{language.text(
					'Dine tips tel saman med spelte resultat. Topp 2 går direkte vidare, og dei 8 beste trearane går også vidare.',
					'Dine tips tel saman med spelte resultat. Topp 2 går direkte vidare, og dei 8 beste trearane går også vidare.',
					'Your picks count together with played results. The top 2 advance directly, and the 8 best third-placed teams advance too.'
				)}
				{#if bestThirds.size === 0}
					<span>
						{' '}
						{language.text(
							'Fyll ut alle gruppene for å projisere beste treere.',
							'Fyll ut alle gruppene for å projisere beste trearar.',
							'Fill every group to project the best third-placed teams.'
						)}
					</span>
				{/if}
			</p>
		{/if}
	{/if}
</div>

<style>
	.gs {
		margin: 0.5rem 0 0.2rem;
	}
	.gs-toggle {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.4rem;
		padding: 0.45rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		color: var(--muted);
		font-weight: 600;
		font-size: 0.82rem;
	}
	:global(.gs .gs-cv) {
		transition: transform 0.15s ease;
	}
	:global(.gs .gs-cv.up) {
		transform: rotate(180deg);
	}
	.gs-tbl {
		width: 100%;
		border-collapse: collapse;
		margin-top: 0.4rem;
		font-size: 0.85rem;
	}
	.gs-tbl th {
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--muted);
		text-align: center;
		padding: 0.25rem 0.4rem;
	}
	.gs-tbl td {
		text-align: center;
		padding: 0.4rem;
		border-top: 1px solid var(--border);
	}
	.gs-tbl .tl {
		text-align: left;
	}
	.tm {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		min-width: 0;
	}
	.nm {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-weight: 600;
	}
	.rk {
		color: var(--muted);
		font-variant-numeric: tabular-nums;
	}
	.pts {
		font-weight: 800;
	}
	tr.adv .rk {
		color: var(--accent);
		font-weight: 800;
	}
	tr.adv td {
		background: color-mix(in srgb, var(--accent) 8%, transparent);
	}
	tr.third .rk {
		color: var(--gold);
	}
	tr.third td {
		background: color-mix(in srgb, var(--gold) 10%, transparent);
	}
	.note {
		margin: 0.5rem 0 0;
		text-align: center;
	}
	.small {
		font-size: 0.78rem;
	}
</style>