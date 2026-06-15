<script lang="ts">
	// Classic user-circle. Renders the avatar image when present (e.g. from
	// Google OAuth later), otherwise initials on a generated colour so users
	// stay visually distinguishable.
	let {
		name,
		src = null,
		size = 36
	}: { name: string; src?: string | null; size?: number } = $props();

	let initials = $derived(
		name
			.trim()
			.split(/\s+/)
			.slice(0, 2)
			.map((p) => p[0]?.toUpperCase() ?? '')
			.join('') || '?'
	);

	// Deterministic hue from the name.
	let hue = $derived(
		[...name].reduce((a, c) => (a * 31 + c.charCodeAt(0)) % 360, 7)
	);
</script>

{#if src}
	<img
		class="avatar"
		{src}
		alt={name}
		style="width:{size}px;height:{size}px"
		referrerpolicy="no-referrer"
	/>
{:else}
	<span
		class="avatar fallback"
		style="width:{size}px;height:{size}px;font-size:{size *
			0.4}px;background:hsl({hue} 55% 32%)"
		aria-label={name}
	>
		{initials}
	</span>
{/if}

<style>
	.avatar {
		display: inline-grid;
		place-items: center;
		border-radius: 50%;
		object-fit: cover;
		border: 2px solid var(--border);
		flex: none;
	}
	.fallback {
		color: var(--accent-fg);
		font-weight: 700;
		letter-spacing: 0.02em;
	}
</style>
