<script lang="ts">
	import { tick } from 'svelte';
	import { Check, Edit3, Send, SmilePlus, Trash2, Undo2, Wifi, WifiOff, X } from '@lucide/svelte';
	import { auth } from '$lib/auth.svelte';
	import { CHAT_EMOJIS, leagueChat, type ChatMessage } from '$lib/chat.svelte';
	import { leagueBadges } from '$lib/leagueBadges.svelte';
	import { language } from '$lib/language.svelte';
	import Avatar from './Avatar.svelte';

	let { leagueId }: { leagueId: string } = $props();

	let draft = $state('');
	let editId = $state('');
	let editDraft = $state('');
	let deleteId = $state('');
	let emojiFor = $state<'composer' | string | null>(null);
	let toolsFor = $state<string | null>(null);
	let listEl = $state<HTMLDivElement | null>(null);
	let composerEl = $state<HTMLTextAreaElement | null>(null);
	let undoMessage = $state<ChatMessage | null>(null);
	let undoTimer: ReturnType<typeof setTimeout> | null = null;
	let deleteTimer: ReturnType<typeof setTimeout> | null = null;
	let deleteTarget = $derived(leagueChat.messages.find((message) => message.id === deleteId) ?? null);

	function onBubbleTap(event: MouseEvent, messageId: string) {
		const target = event.target as HTMLElement | null;
		if (target?.closest('button, textarea, a, input, .reaction')) return;
		toolsFor = toolsFor === messageId ? null : messageId;
	}

	function onBubbleKeydown(event: KeyboardEvent, message: ChatMessage) {
		if (message.deleted) return;
		if (event.key !== 'Enter' && event.key !== ' ') return;
		event.preventDefault();
		toolsFor = toolsFor === message.id ? null : message.id;
	}

	$effect(() => {
		const current = leagueId;
		// Tell the badge store this chat is on screen so badges/toasts stay
		// quiet for it while the user is reading.
		leagueBadges.setActiveChatLeague(current);
		void leagueChat.load(current).then(() => scrollToBottom());
		// Realtime misses events while the tab sleeps; refetch when the user
		// comes back (the fetch also re-marks the chat as read server-side).
		const onVisibilityChange = () => {
			if (document.visibilityState === 'visible') void leagueChat.load(current);
		};
		document.addEventListener('visibilitychange', onVisibilityChange);
		return () => {
			document.removeEventListener('visibilitychange', onVisibilityChange);
			leagueBadges.clearActiveChatLeague(current);
			leagueChat.disconnect();
			clearUndo();
			clearDeleteTimer();
		};
	});

	$effect(() => {
		if (typeof window === 'undefined' || !window.visualViewport) return;
		const onResize = () => scheduleScrollToBottom();
		window.visualViewport.addEventListener('resize', onResize);
		return () => window.visualViewport?.removeEventListener('resize', onResize);
	});

	$effect(() => {
		const count = leagueChat.messages.length;
		if (count > 0) void tick().then(() => scrollToBottom());
	});

	function scrollToBottom() {
		if (!listEl) return;
		listEl.scrollTop = listEl.scrollHeight;
	}

	function scheduleScrollToBottom() {
		requestAnimationFrame(() => requestAnimationFrame(scrollToBottom));
	}

	function growComposer() {
		if (!composerEl) return;
		composerEl.style.height = 'auto';
		composerEl.style.height = `${Math.min(composerEl.scrollHeight, 122)}px`;
	}

	function resetComposer() {
		if (!composerEl) return;
		composerEl.style.height = '';
	}

	function clearUndo() {
		if (undoTimer) clearTimeout(undoTimer);
		undoTimer = null;
		undoMessage = null;
	}

	function showUndo(message: ChatMessage) {
		clearUndo();
		undoMessage = message;
		undoTimer = setTimeout(() => {
			undoMessage = null;
			undoTimer = null;
		}, 30_000);
	}

	function clearDeleteTimer() {
		if (deleteTimer) clearTimeout(deleteTimer);
		deleteTimer = null;
	}

	function cancelDelete() {
		deleteId = '';
		clearDeleteTimer();
	}

	const locale = $derived(language.locale);
	const copy = $derived(
		language.text(
			{
				now: 'nå',
				kicker: 'Liga-chat',
				title: 'Prat med ligaen',
				syncing: 'Oppdaterer',
				loading: 'Laster chat...',
				emptyTitle: 'Ingen meldinger ennå',
				emptyBody: 'Start praten med en kort hilsen eller en reaksjon på tabellen.',
				you: 'Du',
				edited: 'redigert',
				messageOptions: 'Meldingsvalg',
				addReaction: 'Legg til reaksjon',
				edit: 'Rediger',
				delete: 'Slett',
				save: 'Lagre',
				cancel: 'Avbryt',
				deleteConfirm: 'Slette meldingen?',
				deletedMessage: 'Meldingen er slettet',
				deletedOriginal: 'Original melding',
				undoDeleted: 'Melding slettet',
				undo: 'Angre',
				placeholder: 'Skriv en melding...',
				chooseEmoji: 'Velg emoji',
				sending: 'Sender...',
				send: 'Send'
			},
			{
				now: 'no',
				kicker: 'Liga-chat',
				title: 'Prat med ligaen',
				syncing: 'Synkar',
				loading: 'Lastar chat…',
				emptyTitle: 'Ingen meldingar enno',
				emptyBody: 'Start praten med ein kort helsing eller ein reaksjon på tabellen.',
				you: 'Du',
				edited: 'redigert',
				messageOptions: 'Meldingsval',
				addReaction: 'Legg til reaksjon',
				edit: 'Rediger',
				delete: 'Slett',
				save: 'Lagre',
				cancel: 'Avbryt',
				deleteConfirm: 'Slette meldinga?',
				deletedMessage: 'Meldinga er sletta',
				deletedOriginal: 'Original melding',
				undoDeleted: 'Melding sletta',
				undo: 'Angre',
				placeholder: 'Skriv ei melding…',
				chooseEmoji: 'Vel emoji',
				sending: 'Sender…',
				send: 'Send'
			},
			{
				now: 'now',
				kicker: 'League chat',
				title: 'Talk with your league',
				syncing: 'Syncing',
				loading: 'Loading chat…',
				emptyTitle: 'No messages yet',
				emptyBody: 'Start the chat with a quick hello or a reaction to the table.',
				you: 'You',
				edited: 'edited',
				messageOptions: 'Message options',
				addReaction: 'Add reaction',
				edit: 'Edit',
				delete: 'Delete',
				save: 'Save',
				cancel: 'Cancel',
				deleteConfirm: 'Delete message?',
				deletedMessage: 'Message deleted',
				deletedOriginal: 'Original message',
				undoDeleted: 'Message deleted',
				undo: 'Undo',
				placeholder: 'Write a message…',
				chooseEmoji: 'Choose emoji',
				sending: 'Sending…',
				send: 'Send'
			}
		)
	);

	function timeLabel(iso: string) {
		const then = new Date(iso).getTime();
		const diff = Date.now() - then;
		if (!Number.isFinite(then)) return '';
		if (diff < 60_000) return copy.now;
		if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} min`;
		if (diff < 86_400_000) return new Intl.DateTimeFormat(locale, { hour: '2-digit', minute: '2-digit' }).format(then);
		return new Intl.DateTimeFormat(locale, { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' }).format(then);
	}

	async function send() {
		const text = draft.trim();
		if (!text) return;
		await leagueChat.send(text);
		draft = '';
		emojiFor = null;
		await tick();
		resetComposer();
		scrollToBottom();
	}

	function onComposerKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			void send();
		}
	}

	function beginEdit(message: ChatMessage) {
		if (message.deleted) return;
		editId = message.id;
		editDraft = message.text;
		deleteId = '';
		clearDeleteTimer();
		emojiFor = null;
	}

	async function saveEdit() {
		const text = editDraft.trim();
		if (!editId || !text) return;
		await leagueChat.edit(editId, text);
		editId = '';
		editDraft = '';
	}

	function onEditKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			void saveEdit();
		}
		if (event.key === 'Escape') {
			editId = '';
			editDraft = '';
		}
	}

	function askRemove(message: ChatMessage) {
		if (message.deleted) return;
		clearDeleteTimer();
		deleteId = deleteId === message.id ? '' : message.id;
		editId = '';
		emojiFor = null;
		if (deleteId) {
			deleteTimer = setTimeout(() => {
				deleteId = '';
				deleteTimer = null;
			}, 8000);
		}
	}

	async function remove(message: ChatMessage) {
		const deleted = await leagueChat.delete(message.id);
		deleteId = '';
		clearDeleteTimer();
		if (deleted) showUndo(deleted);
	}

	async function react(message: ChatMessage, emoji: string) {
		if (message.deleted) return;
		await leagueChat.toggleReaction(message.id, emoji);
		emojiFor = null;
	}

	async function undoDelete() {
		if (!undoMessage) return;
		const id = undoMessage.id;
		clearUndo();
		await leagueChat.restore(id);
		await tick();
		scrollToBottom();
	}

	function addDraftEmoji(emoji: string) {
		draft += emoji;
		emojiFor = null;
		void tick().then(growComposer);
	}
</script>

<section class="card league-chat" id="chat">
	<div class="chat-head">
		<div>
			<p class="kicker">{copy.kicker}</p>
			<h3>{copy.title}</h3>
		</div>
		<span class="sync" class:live={leagueChat.connected}>
			{#if leagueChat.connected}<Wifi size={15} />{:else}<WifiOff size={15} />{/if}
			{leagueChat.connected ? 'Live' : copy.syncing}
		</span>
	</div>

	{#if leagueChat.error}
		<p class="error">{leagueChat.error}</p>
	{/if}

	<div class="messages" bind:this={listEl}>
		{#if leagueChat.loading && !leagueChat.loaded}
			<p class="muted empty">{copy.loading}</p>
		{:else if leagueChat.messages.length === 0}
			<div class="empty-state">
				<strong>{copy.emptyTitle}</strong>
				<p class="muted">{copy.emptyBody}</p>
			</div>
		{:else}
			<div class="message-stack">
			{#each leagueChat.messages as message (message.id)}
				{@const mine = message.userId === auth.user?.id}
				{@const reactions = leagueChat.reactionSummary(message, auth.user?.id)}
				<article class="message" class:mine>
					<Avatar name={message.user.name} src={message.user.avatarUrl} size={30} />
					<div
						class="bubble"
						class:deleted={message.deleted}
						class:actions-open={emojiFor === message.id || editId === message.id || deleteId === message.id}
						class:tools-open={toolsFor === message.id}
						role="button"
						aria-disabled={message.deleted}
						tabindex={message.deleted ? -1 : 0}
						onclick={(e) => { if (!message.deleted) onBubbleTap(e, message.id); }}
						onkeydown={(e) => onBubbleKeydown(e, message)}
					>
						<div class="meta">
							<strong>{mine ? copy.you : message.user.name}</strong>
							<span>{timeLabel(message.created)}{!message.deleted && message.editedAt ? ` · ${copy.edited}` : ''}</span>
						</div>
						{#if !message.deleted}
							<div class="message-tools" aria-label={copy.messageOptions}>
								<button class="tool-btn" aria-label={copy.addReaction} onclick={() => (emojiFor = emojiFor === message.id ? null : message.id)}>
									<SmilePlus size={14} />
								</button>
								{#if mine && editId !== message.id}
									<button class="tool-btn" aria-label={copy.edit} onclick={() => beginEdit(message)}><Edit3 size={13} /></button>
									<button class="tool-btn danger" aria-label={copy.delete} onclick={() => askRemove(message)}><Trash2 size={13} /></button>
								{/if}
							</div>
						{/if}

						{#if editId === message.id && !message.deleted}
							<textarea class="input edit-input" bind:value={editDraft} rows="3" onkeydown={onEditKeydown}></textarea>
							<div class="edit-actions">
								<button class="icon-btn ok" aria-label={copy.save} onclick={saveEdit}><Check size={16} /></button>
								<button class="icon-btn" aria-label={copy.cancel} onclick={() => { editId = ''; editDraft = ''; }}><X size={16} /></button>
							</div>
						{:else if message.deleted}
							<p class="text deleted-text">{copy.deletedMessage}</p>
							{#if message.origText}
								<p class="admin-original"><strong>{copy.deletedOriginal}:</strong> {message.origText}</p>
							{/if}
						{:else}
							<p class="text">{message.text}</p>
						{/if}

						{#if !message.deleted && reactions.length > 0}
							<div class="reaction-row">
							{#each reactions as reaction (reaction.emoji)}
								<button
									class="reaction"
									class:mine={reaction.mine}
									title={reaction.users.join(', ')}
									onclick={() => react(message, reaction.emoji)}
								>
									<span>{reaction.emoji}</span><b>{reaction.count}</b>
								</button>
							{/each}
							</div>
						{/if}

						{#if !message.deleted && emojiFor === message.id}
							<div class="emoji-panel">
								{#each CHAT_EMOJIS as emoji (emoji)}
									<button onclick={() => react(message, emoji)}>{emoji}</button>
								{/each}
							</div>
						{/if}
					</div>
				</article>
			{/each}
			</div>
		{/if}
	</div>

	{#if deleteTarget && !deleteTarget.deleted}
		<div class="delete-confirm" role="alert">
			<span>{copy.deleteConfirm}</span>
			<div class="delete-actions">
				<button class="mini danger" onclick={() => remove(deleteTarget)}>{copy.delete}</button>
				<button class="mini" onclick={cancelDelete}>{copy.cancel}</button>
			</div>
		</div>
	{/if}

	<div class="composer">
		<div class="composer-input">
			<textarea
				class="input"
				bind:this={composerEl}
				bind:value={draft}
				rows="2"
				maxlength="1000"
				placeholder={copy.placeholder}
				oninput={growComposer}
				onfocus={scheduleScrollToBottom}
				onkeydown={onComposerKeydown}
			></textarea>
			<button class="icon-btn emoji-toggle" aria-label={copy.chooseEmoji} onclick={() => (emojiFor = emojiFor === 'composer' ? null : 'composer')}>
				<SmilePlus size={17} />
			</button>
		</div>
		{#if emojiFor === 'composer'}
			<div class="emoji-panel composer-panel">
				{#each CHAT_EMOJIS as emoji (emoji)}
					<button onclick={() => addDraftEmoji(emoji)}>{emoji}</button>
				{/each}
			</div>
		{/if}
		<button class="btn send" disabled={!draft.trim() || leagueChat.sending} onclick={send}>
			<Send size={17} /> {leagueChat.sending ? copy.sending : copy.send}
		</button>
	</div>

	{#if undoMessage}
		<div class="undo-toast" role="status">
			<span><Undo2 size={16} /> {copy.undoDeleted}</span>
			<button class="mini" onclick={undoDelete}>{copy.undo}</button>
		</div>
	{/if}
</section>

<style>
	.league-chat {
		display: grid;
		gap: 1rem;
	}
	.chat-head {
		display: flex;
		align-items: start;
		justify-content: space-between;
		gap: 1rem;
	}
	.chat-head .kicker,
	.chat-head h3,
	.empty-state p,
	.text {
		margin: 0;
	}
	.sync {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		min-height: 32px;
		padding: 0.35rem 0.65rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		color: var(--muted);
		font-size: 0.8rem;
		font-weight: 700;
	}
	.sync.live {
		color: var(--accent);
		border-color: color-mix(in srgb, var(--accent) 42%, var(--border));
		background: color-mix(in srgb, var(--accent) 10%, transparent);
	}
	.messages {
		display: flex;
		flex-direction: column;
		max-height: min(56vh, 560px);
		min-height: 180px;
		overflow: auto;
		overscroll-behavior: contain;
		padding: 0.15rem 0.15rem 0.2rem;
	}
	.message-stack {
		display: grid;
		gap: 0.55rem;
		margin-top: auto;
	}
	.empty,
	.empty-state {
		align-self: center;
		justify-self: center;
		margin: auto;
		text-align: center;
	}
	.empty-state {
		display: grid;
		gap: 0.25rem;
		padding: 1.8rem 0.5rem;
	}
	.message {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr);
		gap: 0.5rem;
		align-items: start;
	}
	.message.mine {
		grid-template-columns: minmax(0, 1fr) auto;
	}
	.message.mine :global(.avatar) {
		grid-column: 2;
		grid-row: 1;
	}
	.message.mine .bubble {
		grid-column: 1;
		grid-row: 1;
		justify-self: end;
		background: color-mix(in srgb, var(--accent) 13%, var(--surface-2));
	}
	.bubble {
		position: relative;
		display: grid;
		gap: 0.32rem;
		max-width: min(680px, 100%);
		padding: 0.6rem 0.72rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
	}
	.bubble.deleted {
		border-style: dashed;
		background: color-mix(in srgb, var(--surface-2) 78%, transparent);
	}
	.meta {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 0.75rem;
		font-size: 0.85rem;
	}
	.meta span {
		color: var(--muted);
		font-size: 0.75rem;
		white-space: nowrap;
	}
	.message-tools {
		position: absolute;
		top: 0.35rem;
		right: 0.42rem;
		display: inline-flex;
		align-items: center;
		gap: 0.15rem;
		padding: 0.12rem;
		border: 1px solid color-mix(in srgb, var(--border) 72%, transparent);
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--surface) 94%, transparent);
		box-shadow: 0 8px 22px -18px rgba(9, 9, 11, 0.45);
		opacity: 0;
		pointer-events: none;
		transform: translateY(-2px);
		transition: opacity 0.14s ease, transform 0.14s ease;
	}
	.message.mine .message-tools {
		right: auto;
		left: 0.42rem;
	}
	.bubble:hover .message-tools,
	.bubble:focus-within .message-tools,
	.bubble.actions-open .message-tools,
	.bubble.tools-open .message-tools {
		opacity: 1;
		pointer-events: auto;
		transform: translateY(0);
	}
	.tool-btn {
		display: inline-grid;
		place-items: center;
		width: 26px;
		height: 26px;
		border: 0;
		border-radius: 999px;
		background: transparent;
		color: var(--muted);
		cursor: pointer;
	}
	.tool-btn:hover {
		background: var(--surface-2);
		color: var(--text);
	}
	.tool-btn.danger:hover {
		color: var(--danger);
	}
	.text {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		line-height: 1.35;
	}
	.deleted-text {
		color: var(--muted);
		font-style: italic;
	}
	.admin-original {
		margin: 0.1rem 0 0;
		padding: 0.45rem 0.55rem;
		border: 1px solid color-mix(in srgb, var(--warning) 38%, var(--border));
		border-radius: var(--radius-sm);
		background: color-mix(in srgb, var(--warning) 9%, var(--surface));
		color: var(--text);
		font-size: 0.8rem;
		line-height: 1.35;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
	.reaction-row,
	.edit-actions {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		flex-wrap: wrap;
	}
	.delete-confirm {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.4rem;
		padding: 0.45rem;
		border: 1px solid color-mix(in srgb, var(--danger) 34%, var(--border));
		border-radius: var(--radius-sm);
		background: color-mix(in srgb, var(--danger) 7%, var(--surface));
		font-size: 0.78rem;
		font-weight: 700;
	}
	.delete-actions {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		flex-shrink: 0;
	}
	.mini {
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		color: var(--text);
		padding: 0.28rem 0.55rem;
		font: inherit;
		font-size: 0.76rem;
		font-weight: 800;
		cursor: pointer;
	}
	.mini.danger {
		border-color: color-mix(in srgb, var(--danger) 45%, var(--border));
		background: var(--danger);
		color: white;
	}
	.reaction {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		min-height: 24px;
		padding: 0.1rem 0.4rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface);
		color: var(--text);
		font: inherit;
		font-size: 0.76rem;
		cursor: pointer;
	}
	.reaction.mine {
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
		background: color-mix(in srgb, var(--accent) 12%, var(--surface));
	}
	.icon-btn {
		display: inline-grid;
		place-items: center;
		width: 34px;
		height: 34px;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--muted);
		cursor: pointer;
	}
	.icon-btn:hover,
	.reaction:hover {
		border-color: var(--border-strong);
		color: var(--text);
	}
	.icon-btn.ok {
		color: var(--accent);
	}
	.emoji-panel {
		display: flex;
		flex-wrap: wrap;
		gap: 0.3rem;
		padding: 0.45rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
		box-shadow: var(--shadow-pop);
	}
	.emoji-panel button {
		width: 36px;
		height: 36px;
		border: 0;
		border-radius: 999px;
		background: transparent;
		font-size: 1.15rem;
		cursor: pointer;
	}
	.emoji-panel button:hover {
		background: var(--surface-2);
	}
	.composer {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		gap: 0.6rem;
		align-items: end;
	}
	.composer-input {
		position: relative;
	}
	.composer textarea,
	.edit-input {
		min-height: 48px;
		padding-right: 3rem;
		line-height: 1.35;
	}
	.composer textarea {
		max-height: 7.6rem;
		resize: none;
		overflow-y: auto;
	}
	.edit-input {
		resize: vertical;
	}
	.emoji-toggle {
		position: absolute;
		right: 0.45rem;
		bottom: 0.45rem;
	}
	.composer-panel {
		grid-column: 1 / -1;
		box-shadow: none;
	}
	.send {
		width: auto;
		min-height: 48px;
		padding-inline: 1rem;
	}
	.undo-toast {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		padding: 0.65rem 0.75rem;
		border: 1px solid color-mix(in srgb, var(--accent) 38%, var(--border));
		border-radius: var(--radius-sm);
		background: color-mix(in srgb, var(--surface) 92%, transparent);
		box-shadow: var(--shadow-pop);
	}
	.undo-toast span {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.85rem;
		font-weight: 800;
	}
	@media (max-width: 640px) {
		.league-chat {
			padding: 1rem;
		}
		.chat-head {
			align-items: center;
		}
		.messages {
			max-height: 52vh;
		}
		.composer {
			grid-template-columns: 1fr;
		}
		.delete-confirm {
			align-items: stretch;
			flex-direction: column;
		}
		.delete-actions {
			justify-content: flex-end;
		}
		.send {
			width: 100%;
		}
	}

</style>
