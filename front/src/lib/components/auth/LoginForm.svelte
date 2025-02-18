<script lang="ts">
	import { auth } from '$lib/stores/auth';
	import type { LoginRequest } from '$lib/types/auth';

	export let onLoggedIn: () => void;

	let formData: LoginRequest = {
		email: '',
		password: ''
	};

	let error = '';
	let loading = false;

	async function handleSubmit() {
		loading = true;
		error = '';

		try {
			const token = btoa(formData.email + '_' + Date.now());
			const user = {
				id: token,
				email: formData.email,
				nickname: formData.email.split('@')[0]
			};

			auth.login(token, user);
			onLoggedIn();
		} catch (e) {
			error = e instanceof Error ? e.message : 'An error occurred';
		} finally {
			loading = false;
		}
	}
</script>

<form on:submit|preventDefault={handleSubmit} class="space-y-4">
	<div>
		<label for="email" class="block text-sm font-medium text-gray-700">Email</label>
		<input
			type="email"
			id="email"
			bind:value={formData.email}
			required
			class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
			placeholder="Enter your email"
		/>
	</div>

	<div>
		<label for="password" class="block text-sm font-medium text-gray-700">Password</label>
		<input
			type="password"
			id="password"
			bind:value={formData.password}
			required
			class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
			placeholder="Enter your password"
		/>
	</div>

	{#if error}
		<div class="text-sm text-red-500">{error}</div>
	{/if}

	<button
		type="submit"
		disabled={loading}
		class="inline-flex justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50"
	>
		{loading ? 'Logging in...' : 'Login'}
	</button>
</form>
