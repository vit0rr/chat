<script lang="ts">
	import { auth } from '$lib/stores/auth';
	import LoginForm from '$lib/components/auth/LoginForm.svelte';
	import RegisterForm from '$lib/components/chat/RegisterForm.svelte';
	import ChatRoom from '$lib/components/chat/ChatRoom.svelte';
	import type { Room } from '$lib/types/chat';

	let currentRoom: Room | null = null;

	function handleRegistered(event: CustomEvent<Room>) {
		currentRoom = event.detail;
	}
</script>

<div class="min-h-screen bg-gray-50 px-4 py-12 sm:px-6 lg:px-8">
	<div class="mx-auto max-w-3xl">
		{#if !$auth.token}
			<div class="mx-auto max-w-md">
				<LoginForm onLoggedIn={() => {}} />
			</div>
		{:else if !currentRoom}
			<div class="mx-auto max-w-md">
				<div class="mb-8 text-center">
					<h1 class="mb-3 text-4xl font-bold text-gray-900">Chat Room</h1>
					<p class="text-gray-600">Enter a room ID to join a chat room</p>
				</div>

				<div class="rounded-2xl border border-gray-100 bg-white p-8 shadow-lg">
					<RegisterForm on:registered={handleRegistered} />
				</div>
			</div>
		{:else}
			<div class="overflow-hidden rounded-2xl border border-gray-100 bg-white shadow-lg">
				<ChatRoom
					room={currentRoom}
					userId={$auth.user?.id ?? ''}
					nickname={$auth.user?.nickname ?? ''}
				/>
			</div>
		{/if}
	</div>
</div>
