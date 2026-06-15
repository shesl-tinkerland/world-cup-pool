import { House, Volleyball, Telescope, Network, Trophy } from '@lucide/svelte';
import type { Component } from 'svelte';

export interface NavItem {
	href: string;
	labelKey: 'home' | 'matchTips' | 'worldCupTips' | 'bracket' | 'leagues';
	icon: Component;
}

export const navItems: NavItem[] = [
	{ href: '/', labelKey: 'home', icon: House },
	{ href: '/tips', labelKey: 'matchTips', icon: Volleyball },
	{ href: '/forecast', labelKey: 'worldCupTips', icon: Telescope },
	{ href: '/tournament', labelKey: 'bracket', icon: Network },
	{ href: '/leagues', labelKey: 'leagues', icon: Trophy }
];

export function isActive(href: string, path: string): boolean {
	return href === '/' ? path === '/' : path.startsWith(href);
}
