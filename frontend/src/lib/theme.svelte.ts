// Theme rune-store: light / dark / system / worldcup, persisted to localStorage.
// The actual <html data-theme> is set by the inline script in app.html before
// paint (avoids FOUC). This module syncs subsequent user toggles to both
// localStorage and the document, and exposes the current resolved theme.

import { browser } from '$app/environment';

const KEY = 'theme';
type StandardThemeMode = 'light' | 'dark' | 'system';
type ThemeMode = StandardThemeMode | 'worldcup';

function isThemeMode(value: string | null): value is ThemeMode {
	return (
		value === 'light' ||
		value === 'dark' ||
		value === 'system' ||
		value === 'worldcup'
	);
}

function readStored(): ThemeMode {
	if (!browser) return 'worldcup';
	const v = localStorage.getItem(KEY);
	return isThemeMode(v) ? v : 'worldcup';
}

function systemPref(): 'light' | 'dark' {
	if (!browser) return 'dark';
	return matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function apply(mode: ThemeMode) {
	if (!browser) return;
	const root = document.documentElement;
	if (mode === 'system') root.removeAttribute('data-theme');
	else root.setAttribute('data-theme', mode);
	const resolved = mode === 'system' ? systemPref() : mode === 'worldcup' ? 'dark' : mode;
	const meta = document.querySelector('meta[name="theme-color"]');
	if (meta)
		meta.setAttribute(
			'content',
			mode === 'worldcup' ? '#071019' : resolved === 'light' ? '#fafaf9' : '#0b0b0d'
		);
}

const initialMode = readStored();

class ThemeStore {
	mode = $state<ThemeMode>(initialMode);
	lastStandardMode = $state<StandardThemeMode>(initialMode === 'worldcup' ? 'dark' : initialMode);

	get resolved(): 'light' | 'dark' {
		if (this.mode === 'worldcup') return 'dark';
		return this.mode === 'system' ? systemPref() : this.mode;
	}

	get isWorldCup(): boolean {
		return this.mode === 'worldcup';
	}

	set(next: ThemeMode) {
		this.mode = next;
		if (next !== 'worldcup') this.lastStandardMode = next;
		if (browser) {
			if (next === 'system') localStorage.removeItem(KEY);
			else localStorage.setItem(KEY, next);
			apply(next);
		}
	}

	toggleWorldCup() {
		this.set(this.mode === 'worldcup' ? this.lastStandardMode : 'worldcup');
	}

	toggle() {
		this.set(this.resolved === 'dark' ? 'light' : 'dark');
	}
}

export const theme = new ThemeStore();

// Re-apply when the OS preference changes (only matters while mode === 'system').
if (browser) {
	matchMedia('(prefers-color-scheme: light)').addEventListener('change', () => {
		if (theme.mode === 'system') apply('system');
	});
}
