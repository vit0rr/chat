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

<form on:submit|preventDefault={handleSubmit} class="space-y-6">
	<div>
		<label for="room_id" class="mb-2 block text-sm font-medium text-gray-700">Room ID</label>
		<input
			type="text"
			id="room_id"
			bind:value={formData.room_id}
			required
			class="block w-full rounded-xl px-4 py-3 text-gray-900 ring-1 ring-inset ring-gray-200 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm"
			placeholder="Enter room ID"
		/>
	</div>

	<div>
		<label for="nickname" class="mb-2 block text-sm font-medium text-gray-700">Nickname</label>
		<input
			type="text"
			id="nickname"
			bind:value={formData.nickname}
			required
			class="block w-full rounded-xl px-4 py-3 text-gray-900 ring-1 ring-inset ring-gray-200 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm"
			placeholder="Enter your nickname"
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
		{loading ? 'Joining...' : 'Join Room'}
	</button>
</form>
