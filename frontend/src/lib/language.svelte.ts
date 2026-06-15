import { browser } from '$app/environment';
import { pb } from './pb';

export type LanguageCode = 'nb' | 'nn' | 'en';

const STORAGE_KEY = 'language';
const DEFAULT_LANGUAGE: LanguageCode = 'en';
const LANGUAGE_ORDER: LanguageCode[] = ['en', 'nb', 'nn'];

export function isLanguageCode(value: unknown): value is LanguageCode {
	return value === 'nb' || value === 'nn' || value === 'en';
}

function readAuthLanguage(): LanguageCode | null {
	const value = pb.authStore.record?.language;
	return isLanguageCode(value) ? value : null;
}

function readStoredLanguage(): LanguageCode {
	if (!browser) return DEFAULT_LANGUAGE;
	const authLanguage = readAuthLanguage();
	if (authLanguage) return authLanguage;
	const stored = localStorage.getItem(STORAGE_KEY);
	return isLanguageCode(stored) ? stored : DEFAULT_LANGUAGE;
}

class LanguageStore {
	code = $state<LanguageCode>(readStoredLanguage());
	private persisting = false;

	constructor() {
		pb.authStore.onChange(() => {
			void this.syncFromAuth();
		});
		if (browser) {
			localStorage.setItem(STORAGE_KEY, this.code);
			queueMicrotask(() => {
				void this.syncFromAuth();
			});
		}
	}

	get resolved() {
		return this.code;
	}

	get locale() {
		// `nn-NO` falls back inconsistently in some browsers; `no-NO`
		// keeps Norwegian date/time formatting stable while UI copy chooses nb/nn.
		return this.code === 'en' ? 'en-US' : 'no-NO';
	}

	get isEnglish() {
		return this.code === 'en';
	}

	get isBokmal() {
		return this.code === 'nb';
	}

	get isNynorsk() {
		return this.code === 'nn';
	}

	text<T>(nb: T, nn: T, en: T): T {
		if (this.code === 'en') return en;
		if (this.code === 'nn') return nn;
		return nb;
	}

	private async syncFromAuth() {
		const authLanguage = readAuthLanguage();
		if (authLanguage) {
			this.code = authLanguage;
			if (browser) localStorage.setItem(STORAGE_KEY, authLanguage);
			return;
		}
		if (!pb.authStore.isValid || !pb.authStore.record) return;
		await this.persist(this.code);
	}

	private async persist(next: LanguageCode) {
		if (!browser || this.persisting || !pb.authStore.isValid || !pb.authStore.record) {
			return;
		}
		const record = pb.authStore.record;
		if (record.language === next) return;
		this.persisting = true;
		try {
			await pb.collection('users').update(record.id, { language: next });
			await pb.collection('users').authRefresh();
		} catch {
			// Keep the local preference even if syncing the user record fails.
		} finally {
			this.persisting = false;
		}
	}

	set(next: LanguageCode) {
		this.code = next;
		if (browser) localStorage.setItem(STORAGE_KEY, next);
		void this.persist(next);
	}

	toggle() {
		const currentIndex = LANGUAGE_ORDER.indexOf(this.code);
		this.set(LANGUAGE_ORDER[(currentIndex + 1) % LANGUAGE_ORDER.length]);
	}
}

export const language = new LanguageStore();
