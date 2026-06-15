class SearchNavState {
	token = $state(0);

	bump() {
		this.token = Date.now();
	}
}

export const searchNav = new SearchNavState();