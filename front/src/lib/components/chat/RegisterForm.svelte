<script lang="ts">
	import { enhance } from '$app/forms';
	import type { RegisterUserRequest, Room } from '$lib/types/chat';
	import { createEventDispatcher } from 'svelte';
	import { auth } from '$lib/stores/auth';

	const dispatch = createEventDispatcher<{
		registered: Room;
	}>();

	let formData: RegisterUserRequest = {
		room_id: '',
		user_id: $auth.user?.id || '',
		nickname: $auth.user?.nickname || ''
	};

	let error = '';
	let loading = false;

	async function handleSubmit() {
		loading = true;
		error = '';

		try {
			const response = await fetch('http://localhost:8080/api/register-user', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${$auth.token}`
				},
				body: JSON.stringify({
					room_id: formData.room_id,
					user_id: $auth.user?.id,
					nickname: formData.nickname
				})
			});

			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error || 'Failed to register');
			}

			dispatch('registered', data);
		} catch (e) {
			error = e instanceof Error ? e.message : 'An error occurred';
		} finally {
			loading = false;
		}
	}
</script>

<form on:submit|preventDefault={handleSubmit} class="space-y-4">
	<div>
		<label for="room_id" class="block text-sm font-medium text-gray-700"> Room ID </label>
		<input
			type="text"
			id="room_id"
			bind:value={formData.room_id}
			required
			class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
			placeholder="Enter room ID"
		/>
	</div>

	<div>
		<label for="nickname" class="block text-sm font-medium text-gray-700"> Nickname </label>
		<input
			type="text"
			id="nickname"
			bind:value={formData.nickname}
			required
			class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
			placeholder="Enter your nickname"
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
		{loading ? 'Joining...' : 'Join Room'}
	</button>
</form>
