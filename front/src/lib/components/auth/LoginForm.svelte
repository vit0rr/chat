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

<div class="w-full max-w-md p-8">
	<div class="mb-8 text-center">
		<h1 class="mb-3 text-4xl font-bold text-gray-900">Login</h1>
		<p class="text-gray-600">Please login to continue</p>
	</div>

	<div class="rounded-2xl border border-gray-100 bg-white p-8 shadow-lg">
		<form on:submit|preventDefault={handleSubmit} class="space-y-6">
			<div>
				<label for="email" class="mb-2 block text-sm font-medium text-gray-700">Email</label>
				<input
					type="email"
					id="email"
					bind:value={formData.email}
					required
					class="block w-full rounded-xl px-4 py-3 text-gray-900 ring-1 ring-inset ring-gray-200 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm"
					placeholder="Enter your email"
				/>
			</div>

			<div>
				<label for="password" class="mb-2 block text-sm font-medium text-gray-700">Password</label>
				<input
					type="password"
					id="password"
					bind:value={formData.password}
					required
					class="block w-full rounded-xl px-4 py-3 text-gray-900 ring-1 ring-inset ring-gray-200 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm"
					placeholder="Enter your password"
				/>
			</div>

			{#if error}
				<div class="rounded-lg bg-red-50 px-4 py-2 text-sm text-red-600">{error}</div>
			{/if}

			<button
				type="submit"
				disabled={loading}
				class="w-full rounded-xl bg-indigo-600 px-4 py-3 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-600 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
			>
				{loading ? 'Logging in...' : 'Login'}
			</button>
		</form>
	</div>
</div>
