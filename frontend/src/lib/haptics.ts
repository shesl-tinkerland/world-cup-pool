export function vibrate(ms: number = 10) {
	if (typeof navigator !== 'undefined' && navigator.vibrate) {
		try {
			navigator.vibrate(ms);
		} catch {
			// ignore on unsupported devices
		}
	}
}
