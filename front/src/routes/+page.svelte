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
				<div class="text-center">
					<h1 class="text-3xl font-bold">Login</h1>
					<p class="mt-2 text-gray-600">Please login to continue</p>
				</div>

				<div class="mt-8 bg-white px-4 py-8 shadow sm:rounded-lg sm:px-10">
					<LoginForm onLoggedIn={() => {}} />
				</div>
			</div>
		{:else if !currentRoom}
			<div class="mx-auto max-w-md">
				<div class="text-center">
					<h1 class="text-3xl font-bold">Chat Room Registration/Login</h1>
					<p class="mt-2 text-gray-600">Enter a room ID to join a chat room</p>
				</div>

				<div class="mt-8 bg-white px-4 py-8 shadow sm:rounded-lg sm:px-10">
					<RegisterForm on:registered={handleRegistered} />
				</div>
			</div>
		{:else}
			<ChatRoom
				room={currentRoom}
				userId={$auth.user?.id ?? ''}
				nickname={$auth.user?.nickname ?? ''}
			/>
		{/if}
	</div>
</div>
