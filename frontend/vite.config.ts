import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

const apiOrigin = process.env.VITE_API_ORIGIN ?? 'http://127.0.0.1:8091';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		// During `npm run dev`, proxy API/auth/files to the local PocketBase
		// server on the isolated test port so prod :8090 is never hit by accident.
		proxy: {
			'/api': apiOrigin,
			'/_': apiOrigin
		}
	}
});
