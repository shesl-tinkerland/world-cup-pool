import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
	collection: vi.fn(),
	send: vi.fn(),
	create: vi.fn(),
	authWithPassword: vi.fn(),
	authWithOAuth2: vi.fn(),
	requestPasswordReset: vi.fn(),
	confirmPasswordReset: vi.fn(),
	onChangeHandlers: [] as Array<() => void>,
	authStore: (() => {
		const authStore = {
			isValid: false,
			record: null as null | {
				id: string;
				name?: string;
				email: string;
				avatar?: string | null;
			},
			onChange: vi.fn((cb: () => void) => {
				mocks.onChangeHandlers.push(cb);
				return () => {
					const idx = mocks.onChangeHandlers.indexOf(cb);
					if (idx >= 0) mocks.onChangeHandlers.splice(idx, 1);
				};
			}),
			clear: vi.fn(() => {
				authStore.isValid = false;
				authStore.record = null;
				for (const handler of mocks.onChangeHandlers) handler();
			})
		};

		return authStore;
	})()
}));

vi.mock('./pb', () => ({
	pb: {
		authStore: mocks.authStore,
		send: mocks.send,
		files: { getURL: vi.fn() },
		collection: (name: string) => {
			mocks.collection(name);
			return {
				create: mocks.create,
				authWithPassword: mocks.authWithPassword,
				authWithOAuth2: mocks.authWithOAuth2,
				requestPasswordReset: mocks.requestPasswordReset,
				confirmPasswordReset: mocks.confirmPasswordReset
			};
		}
	}
}));

import { auth } from './auth.svelte';

describe('auth password reset', () => {
	beforeEach(() => {
		mocks.collection.mockClear();
		mocks.create.mockReset();
			mocks.send.mockReset();
		mocks.authWithPassword.mockReset();
		mocks.authWithOAuth2.mockReset();
		mocks.requestPasswordReset.mockReset();
		mocks.confirmPasswordReset.mockReset();
			mocks.authStore.clear.mockClear();
		mocks.authStore.isValid = false;
		mocks.authStore.record = null;
		for (const handler of mocks.onChangeHandlers) handler();
		localStorage.clear();
	});

	it('syncs user state immediately after password login completes', async () => {
		mocks.authWithPassword.mockImplementationOnce(async () => {
			mocks.authStore.isValid = true;
			mocks.authStore.record = {
				id: 'user-1',
				name: 'Test User',
				email: 'test@example.com',
				avatar: null
			};
		});

		await auth.login('test@example.com', 'secret');

		expect(mocks.collection).toHaveBeenCalledWith('users');
		expect(mocks.authWithPassword).toHaveBeenCalledWith(
			'test@example.com',
			'secret'
		);
		expect(auth.user?.email).toBe('test@example.com');
	});

	it('falls back to normalized email when password login is case-sensitive', async () => {
		mocks.authWithPassword
			.mockRejectedValueOnce(new Error('invalid credentials'))
			.mockImplementationOnce(async () => {
				mocks.authStore.isValid = true;
				mocks.authStore.record = {
					id: 'user-1',
					name: 'Test User',
					email: 'test@example.com',
					avatar: null
				};
			});

		await auth.login(' Test@Example.COM ', 'secret');

		expect(mocks.authWithPassword).toHaveBeenNthCalledWith(
			1,
			'Test@Example.COM',
			'secret'
		);
		expect(mocks.authWithPassword).toHaveBeenNthCalledWith(
			2,
			'test@example.com',
			'secret'
		);
		expect(auth.user?.email).toBe('test@example.com');
	});

	it('syncs user state immediately after Google login completes', async () => {
		mocks.authWithOAuth2.mockImplementationOnce(async () => {
			mocks.authStore.isValid = true;
			mocks.authStore.record = {
				id: 'user-2',
				name: 'Google User',
				email: 'google@example.com',
				avatar: null
			};
		});

		await auth.loginGoogle();

		expect(mocks.collection).toHaveBeenCalledWith('users');
		expect(mocks.authWithOAuth2).toHaveBeenCalledWith({ provider: 'google' });
		expect(auth.user?.email).toBe('google@example.com');
	});

	it('requests a PocketBase password reset email for users', async () => {
		mocks.requestPasswordReset.mockResolvedValueOnce(true);

		await auth.requestPasswordReset(' Test@Example.COM ');

		expect(mocks.collection).toHaveBeenCalledWith('users');
		expect(mocks.requestPasswordReset).toHaveBeenCalledWith('test@example.com');
	});

	it('confirms a reset token with the new password', async () => {
		mocks.confirmPasswordReset.mockResolvedValueOnce(true);

		await auth.confirmPasswordReset('reset-token', 'new-password', 'new-password');

		expect(mocks.collection).toHaveBeenCalledWith('users');
		expect(mocks.confirmPasswordReset).toHaveBeenCalledWith(
			'reset-token',
			'new-password',
			'new-password'
		);
	});

	it('deletes the signed-in account via the custom API and clears auth', async () => {
		mocks.authStore.isValid = true;
		mocks.authStore.record = {
			id: 'user-3',
			name: 'Delete Me',
			email: 'delete@example.com',
			avatar: null
		};
		for (const handler of mocks.onChangeHandlers) handler();
		mocks.send.mockResolvedValueOnce(undefined);

		await auth.deleteAccount();

		expect(mocks.send).toHaveBeenCalledWith('/api/account', { method: 'DELETE' });
		expect(mocks.authStore.clear).toHaveBeenCalledTimes(1);
		expect(auth.user).toBeNull();
	});

	it('stores the selected language when registering a new user', async () => {
		localStorage.setItem('language', 'nn');
		mocks.create.mockResolvedValueOnce({ id: 'user-4' });
		mocks.authWithPassword.mockImplementationOnce(async () => {
			mocks.authStore.isValid = true;
			mocks.authStore.record = {
				id: 'user-4',
				name: 'New User',
				email: 'new@example.com',
				avatar: null
			};
		});

		await auth.register('New User', ' New@Example.COM ', 'secret');

		expect(mocks.collection).toHaveBeenCalledWith('users');
		expect(mocks.create).toHaveBeenCalledWith({
			name: 'New User',
			email: 'new@example.com',
			language: 'nn',
			password: 'secret',
			passwordConfirm: 'secret'
		});
	});
});
