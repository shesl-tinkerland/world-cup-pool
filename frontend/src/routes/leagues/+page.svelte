<script lang="ts">
	import { api, type LeagueSummary } from '$lib/api';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { leagueBadges } from '$lib/leagueBadges.svelte';
	import { language } from '$lib/language.svelte';
	import PendingInvites from '$lib/components/PendingInvites.svelte';
	import { ArrowRight, Crown, Globe2, LogIn, MessageCircle, Plus, Users } from '@lucide/svelte';

	let leagues = $state<LeagueSummary[]>([]);
	let loaded = $state(false);
	let newName = $state('');
	let joinCode = $state('');
	let error = $state('');
	let busy = $state(false);

	async function load() {
		try {
			leagues = (await api.myLeagues()).leagues;
		} catch {
			/* ignore */
		} finally {
			loaded = true;
		}
	}
	$effect(() => {
		load();
	});

	onMount(() => {
		leagueBadges.start();
		void leagueBadges.load(true);
		return () => leagueBadges.stop();
	});

	async function create(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			const r = await api.createLeague(newName);
			newName = '';
			goto(`/leagues/${r.id}`);
		} catch {
			error = language.text('Kunne ikke opprette liga.', 'Kunne ikkje opprette liga.', 'Could not create league.');
		} finally {
			busy = false;
		}
	}

	async function join(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			const r = await api.joinLeague(joinCode);
			joinCode = '';
			goto(`/leagues/${r.id}`);
		} catch {
			error = language.text('Ugyldig invitasjonskode.', 'Ugyldig invitasjonskode.', 'Invalid invite code.');
		} finally {
			busy = false;
		}
	}

	function roleLabel(league: LeagueSummary) {
		if (league.inviteCode === 'GLOBAL') return 'Global';
		return league.role === 'owner'
		? language.text('Eier', 'Eigar', 'Owner')
			: language.text('Medlem', 'Medlem', 'Member');
	}

	function unreadChatLabel(count: number) {
		return count === 1
			? language.text('Ny chat', 'Ny chat', 'New chat')
			: language.text(`${count} nye chatter`, `${count} nye chattar`, `${count} new chats`);
	}
</script>

<header class="league-hero">
	<p class="kicker">{language.text('Spill mot vennene dine', 'Spel mot venene dine', 'Play against your friends')}</p>
	<h1>{language.text('Ligaer', 'Ligaer', 'Leagues')}</h1>
	<p class="muted">{language.text('Velg liga, se tabellen og hopp rett til chat.', 'Vel ei liga, sjå tabellen og hopp rett til chat.', 'Pick a league, see the table, and jump straight to chat.')}</p>
</header>

<PendingInvites compact />

<section class="league-section">
	<div class="section-head">
		<div>
			<p class="kicker">{language.text('Oversikt', 'Oversikt', 'Overview')}</p>
			<h2>{language.text('Ligaene dine', 'Ligaene dine', 'Your leagues')}</h2>
		</div>
		{#if loaded}<span class="count-pill">{leagues.length}</span>{/if}
	</div>
	{#if !loaded}
		<div class="league-grid">
			<div class="league-tile skeleton"></div>
			<div class="league-tile skeleton hide-mobile"></div>
		</div>
	{:else if leagues.length === 0}
		<div class="empty-state">
			<strong>{language.text('Ingen ligaer ennå', 'Ingen ligaer enno', 'No leagues yet')}</strong>
			<p class="muted">{language.text('Opprett en liga eller bli med med invitasjonskode.', 'Opprett ei liga eller bli med med invitasjonskode.', 'Create a league or join with an invite code.')}</p>
		</div>
	{:else}
		<div class="league-grid">
			{#each leagues as league (league.id)}
				{@const unreadChat = leagueBadges.unreadForLeague(league.id)}
				<a
					class="league-tile"
					class:has-chat={unreadChat > 0}
					class:global={league.inviteCode === 'GLOBAL'}
					href={`/leagues/${league.id}`}
				>
					<span class="league-icon">
						{#if league.inviteCode === 'GLOBAL'}<Globe2 size={20} />
						{:else if league.role === 'owner'}<Crown size={20} />
						{:else}<Users size={20} />{/if}
					</span>
					<span class="league-main">
						<span class="league-topline">
							<b>{league.name}</b>
							<i>{roleLabel(league)}</i>
						</span>
						<span class="league-meta">
							<span><Users size={14} /> {league.members} {language.text(league.members === 1 ? 'medlem' : 'medlemmer', league.members === 1 ? 'medlem' : 'medlemer', league.members === 1 ? 'member' : 'members')}</span>
							{#if league.inviteCode && league.inviteCode !== 'GLOBAL'}
								<span>{language.text('Kode', 'Kode', 'Code')} {league.inviteCode}</span>
							{:else if league.private}
								<span>{language.text('Privat kode', 'Privat kode', 'Private code')}</span>
							{/if}
						</span>
						{#if unreadChat > 0}
							<span class="chat-alert">
								<MessageCircle size={14} />
								{unreadChatLabel(unreadChat)}
							</span>
						{/if}
					</span>
					<span class="go"><ArrowRight size={18} /></span>
				</a>
			{/each}
		</div>
	{/if}
</section>

<div class="action-grid">
	<section class="card action-card">
		<div class="action-title">
			<span class="action-icon"><Plus size={18} /></span>
			<div>
				<h3>{language.text('Opprett liga', 'Opprett liga', 'Create league')}</h3>
				<p class="muted">{language.text('Start en ny privat konkurranse.', 'Start ei ny privat tevling.', 'Start a new private competition.')}</p>
			</div>
		</div>
		<form onsubmit={create}>
			<div class="field">
				<input class="input" placeholder={language.text('Liganavn', 'Liganamn', 'League name')} bind:value={newName} required />
			</div>
			<button class="btn" disabled={busy || !newName.trim()}>{language.text('Opprett', 'Opprett', 'Create')}</button>
		</form>
	</section>

	<section class="card action-card">
		<div class="action-title">
			<span class="action-icon secondary"><LogIn size={18} /></span>
			<div>
				<h3>{language.text('Bli med', 'Bli med', 'Join')}</h3>
				<p class="muted">{language.text('Lim inn koden fra invitasjonen.', 'Lim inn koden frå invitasjonen.', 'Paste the code from the invite.')}</p>
			</div>
		</div>
		<form onsubmit={join}>
			<div class="field">
				<input
					class="input code"
					placeholder="INVITE CODE"
					bind:value={joinCode}
					required
				/>
			</div>
			<button class="btn secondary" disabled={busy || !joinCode.trim()}>{language.text('Bli med', 'Bli med', 'Join')}</button>
		</form>
	</section>
</div>

{#if error}<p class="error">{error}</p>{/if}

<style>
	.league-hero {
		display: grid;
		gap: 0.35rem;
		margin: 0.2rem 0 1rem;
	}
	.league-hero h1,
	.section-head h2,
	.action-title h3 {
		margin: 0;
	}
	.muted {
		margin: 0;
	}
	.league-section {
		display: grid;
		gap: 0.75rem;
	}
	.section-head {
		display: flex;
		align-items: end;
		justify-content: space-between;
		gap: 1rem;
	}
	.section-head h2 {
		font-size: 1.2rem;
	}
	.count-pill {
		display: grid;
		place-items: center;
		min-width: 2.1rem;
		height: 2.1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface);
		font-family: var(--font-mono);
		font-weight: 800;
		color: var(--text);
	}
	.league-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 0.75rem;
	}
	.league-tile {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 0.8rem;
		min-height: 104px;
		padding: 1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--text);
		box-shadow: var(--shadow-tile);
		transition: transform 0.16s ease, border-color 0.16s ease, box-shadow 0.16s ease;
	}
	.league-tile:hover,
	.league-tile:focus-visible {
		border-color: color-mix(in srgb, var(--accent) 38%, var(--border));
		box-shadow: var(--shadow-pop);
		transform: translateY(-2px);
		outline: none;
	}
	.league-tile.global {
		background: color-mix(in srgb, var(--accent) 6%, var(--surface));
	}
	.league-tile.has-chat {
		border-color: color-mix(in srgb, var(--live) 34%, var(--border));
		background:
			linear-gradient(135deg, color-mix(in srgb, var(--live) 8%, transparent), transparent 52%),
			var(--surface);
	}
	.league-icon,
	.go,
	.action-icon {
		display: grid;
		place-items: center;
		border-radius: 14px;
		background: var(--surface-2);
		color: var(--muted);
	}
	.league-icon {
		width: 44px;
		height: 44px;
	}
	.league-tile.global .league-icon {
		background: color-mix(in srgb, var(--accent) 14%, var(--surface-2));
		color: var(--accent);
	}
	.league-main {
		display: grid;
		gap: 0.45rem;
		min-width: 0;
	}
	.league-topline {
		display: grid;
		gap: 0.35rem;
	}
	.league-topline b {
		font-size: 1.05rem;
		line-height: 1.1;
		overflow-wrap: anywhere;
	}
	.league-topline i {
		width: fit-content;
		padding: 0.16rem 0.45rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		color: var(--muted);
		font-style: normal;
		font-size: 0.7rem;
		font-weight: 800;
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}
	.league-meta {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem 0.6rem;
		color: var(--muted);
		font-size: 0.82rem;
		font-weight: 650;
	}
	.league-meta span {
		display: inline-flex;
		align-items: center;
		gap: 0.28rem;
	}
	.chat-alert {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		width: fit-content;
		margin-top: 0.05rem;
		padding: 0.22rem 0.5rem;
		border-radius: var(--radius-pill);
		background: var(--live);
		color: var(--bg);
		font-size: 0.74rem;
		font-weight: 850;
		box-shadow:
			0 0 0 3px color-mix(in srgb, var(--live) 14%, transparent),
			0 10px 20px -16px color-mix(in srgb, var(--live) 80%, transparent);
	}
	.chat-alert :global(svg) {
		flex: 0 0 auto;
	}
	.go {
		width: 34px;
		height: 34px;
		border-radius: var(--radius-pill);
		transition: transform 0.16s ease, color 0.16s ease, background 0.16s ease;
	}
	.league-tile:hover .go,
	.league-tile:focus-visible .go {
		background: var(--text);
		color: var(--bg);
		transform: translateX(2px);
	}
	.empty-state {
		padding: 1rem;
		border: 1px dashed var(--border-strong);
		border-radius: var(--radius);
		background: var(--surface);
	}
	.empty-state strong {
		display: block;
		margin-bottom: 0.25rem;
	}
	.skeleton {
		min-height: 104px;
		background: linear-gradient(90deg, var(--surface), var(--surface-2), var(--surface));
		background-size: 200% 100%;
		animation: pulse 1.4s ease-in-out infinite;
	}
	.action-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: 0.85rem;
		margin-top: 0.95rem;
	}
	.action-card {
		display: grid;
		gap: 1rem;
	}
	.action-title {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	.action-title .muted {
		font-size: 0.86rem;
	}
	.action-icon {
		width: 42px;
		height: 42px;
		background: color-mix(in srgb, var(--accent) 12%, var(--surface-2));
		color: var(--accent);
	}
	.action-icon.secondary {
		background: color-mix(in srgb, var(--accent-2) 12%, var(--surface-2));
		color: var(--accent-2);
	}
	.field {
		margin-bottom: 0.65rem;
	}
	.code {
		text-transform: uppercase;
		letter-spacing: 0.2em;
		font-weight: 700;
	}
	@keyframes pulse {
		from { background-position: 200% 0; }
		to { background-position: -200% 0; }
	}
	@media (max-width: 560px) {
		.league-grid,
		.action-grid {
			grid-template-columns: minmax(0, 1fr);
		}
		.league-tile {
			min-height: 96px;
			padding: 0.9rem;
			gap: 0.7rem;
		}
		.league-icon {
			width: 40px;
			height: 40px;
		}
		.go {
			width: 31px;
			height: 31px;
		}
		.hide-mobile {
			display: none;
		}
	}
</style>
