// Shared PocketBase mock used by the store / api tests. Each test file calls
// `installPbMock()` in a `beforeEach` and supplies the in-memory dataset it
// wants (teams, matches, etc.). Keeps the tests free of network calls.

import { vi } from 'vitest';

export interface MockDb {
	teams: any[];
	matches: any[];
	tournament_groups: any[];
	tips: any[];
	forecasts: any[];
	send: Record<string, (opts: { method?: string; body?: any }) => any>;
}

export function makeDb(over: Partial<MockDb> = {}): MockDb {
	return {
		teams: [],
		matches: [],
		tournament_groups: [],
		tips: [],
		forecasts: [],
		send: {},
		...over
	};
}

export function installPbMock(db: MockDb) {
	const calls: { kind: string; args: any[] }[] = [];
	const collection = (name: string) => ({
		getFullList: vi.fn(async () => {
			calls.push({ kind: `getFullList:${name}`, args: [] });
			return (db as any)[name] ?? [];
		}),
		create: vi.fn(async (data: any) => {
			calls.push({ kind: `create:${name}`, args: [data] });
			const id = 'rec_' + Math.random().toString(36).slice(2, 9);
			const row = { id, ...data };
			(db as any)[name] = [...((db as any)[name] ?? []), row];
			return row;
		}),
		update: vi.fn(async (id: string, data: any) => {
			calls.push({ kind: `update:${name}`, args: [id, data] });
			return { id, ...data };
		})
	});
	const send = vi.fn(async (path: string, opts: any = {}) => {
		calls.push({ kind: `send:${path}`, args: [opts] });
		const fn = db.send[path];
		if (!fn) throw new Error(`No mock for ${path}`);
		return fn(opts);
	});
	vi.doMock('../lib/pb', () => ({ pb: { collection, send } }));
	vi.doMock('../pb', () => ({ pb: { collection, send } }));
	return { calls };
}
