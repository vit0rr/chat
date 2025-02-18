<script lang="ts">
	import type { Room } from '$lib/types/chat';
	import { onMount, onDestroy } from 'svelte';
	import { fetchMessages } from '$lib/utils/chat';
	import { auth } from '$lib/stores/auth';

	export let room: Room;
	export let userId: string;
	export let nickname: string;

	let messages: any[] = [];
	let messageInput = '';
	let ws: WebSocket;
	let error = '';
	let connected = false;
	let loading = false;
	let currentPage = 1;
	let hasMore = true;
	let messageContainer: HTMLDivElement;
	let isLocked = false;

	onMount(async () => {
		await loadInitialMessages();
		connectWebSocket();
	});

	onDestroy(() => {
		if (ws) {
			ws.close();
		}
	});

	async function loadInitialMessages() {
		try {
			loading = true;
			const initialMessages = await fetchMessages(room.id ?? '', 1, $auth.token ?? '');
			messages = initialMessages;
			hasMore = initialMessages.length === 50;

			setTimeout(() => {
				messageContainer.scrollTop = messageContainer.scrollHeight;
			}, 0);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load messages';
		} finally {
			loading = false;
		}
	}

	async function loadMoreMessages() {
		if (loading || !hasMore) return;

		try {
			loading = true;
			const olderMessages = await fetchMessages(room.id ?? '', currentPage + 1, $auth.token ?? '');
			if (olderMessages.length > 0) {
				messages = [...messages, ...olderMessages];
				currentPage += 1;
				hasMore = olderMessages.length === 50;
			} else {
				hasMore = false;
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load more messages';
		} finally {
			loading = false;
		}
	}

	function handleScroll(event: Event) {
		const target = event.target as HTMLDivElement;
		if (target.scrollTop <= 100 && !loading && hasMore) {
			loadMoreMessages();
		}
	}

	function connectWebSocket() {
		const wsUrl = `ws://localhost:8080/api/ws?room_id=${room.id}&user_id=${userId}&nickname=${nickname}&token=${$auth.token}`;
		ws = new WebSocket(wsUrl);

		ws.onopen = () => {
			connected = true;
			error = '';
		};

		ws.onmessage = handleWebSocketMessage;

		ws.onerror = () => {
			error = 'WebSocket error occurred';
			connected = false;
		};

		ws.onclose = () => {
			connected = false;
			error = 'WebSocket connection closed';
		};
	}

	async function toggleRoomLock() {
		try {
			const response = await fetch('/api/rooms/lock', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${$auth.token}`
				},
				body: JSON.stringify({
					room_id: room.id,
					user_id: userId
				})
			});

			if (!response.ok) {
				throw new Error('Failed to toggle room lock');
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to toggle room lock';
		}
	}

	function handleWebSocketMessage(event: MessageEvent) {
		const message = JSON.parse(event.data);
		if (message.type === 'system') {
			if (message.content.includes('locked by')) {
				isLocked = true;
				room.lockedBy = message.content.includes(nickname) ? userId : 'other';
			} else if (message.content.includes('unlocked by')) {
				isLocked = false;
				room.lockedBy = '';
			}
		}
		messages = [...messages, message];
		setTimeout(() => {
			messageContainer.scrollTop = messageContainer.scrollHeight;
		}, 0);
	}

	function canSendMessage() {
		return connected && (!isLocked || room.lockedBy === userId);
	}

	function sendMessage() {
		if (!messageInput.trim() || !connected || (isLocked && room.lockedBy !== userId)) return;

		const message = {
			type: 'text',
			content: messageInput,
			room_id: room.id,
			sender_id: userId,
			nickname: nickname
		};

		ws.send(JSON.stringify(message));
		messageInput = '';
	}
</script>

<div class="flex h-[600px] flex-col rounded-lg border border-gray-200 bg-white">
	<div class="flex items-center justify-between border-b border-gray-200 p-4">
		<h2 class="text-lg font-semibold text-gray-900">Chat Room</h2>
		<button
			on:click={toggleRoomLock}
			disabled={!connected}
			class="rounded-md {isLocked && room.lockedBy === userId
				? 'bg-green-600 hover:bg-green-700'
				: 'bg-red-600 hover:bg-red-700'} px-4 py-2 text-sm text-white disabled:opacity-50"
		>
			{#if isLocked}
				{room.lockedBy === userId ? 'Unlock Room' : 'Locked by another user'}
			{:else}
				Lock Room
			{/if}
		</button>
	</div>

	<div
		bind:this={messageContainer}
		on:scroll={handleScroll}
		class="flex-1 space-y-4 overflow-y-auto p-4"
	>
		{#if loading && currentPage === 1}
			<div class="text-center text-gray-500">Loading messages...</div>
		{/if}

		{#if hasMore}
			<div class="text-center text-sm text-gray-500">
				{loading ? 'Loading more messages...' : 'Scroll up to load more'}
			</div>
		{/if}

		{#each messages as message}
			<div
				class="flex {message.sender_id === userId || message.fromUserId === userId
					? 'justify-end'
					: 'justify-start'}"
			>
				<div
					class="max-w-[70%] rounded-lg p-3 {message.sender_id === userId ||
					message.fromUserId === userId
						? 'bg-indigo-100'
						: 'bg-gray-100'}"
				>
					{#if message.type === 'system'}
						<p class="text-sm italic text-gray-600">{message.content}</p>
					{:else}
						<p class="text-sm font-semibold text-gray-700">{message.nickname}</p>
						<p class="text-gray-900">{message.content || message.message}</p>
						<p class="mt-1 text-xs text-gray-500">
							{new Date(message.timestamp || message.createdAt).toLocaleTimeString()}
						</p>
					{/if}
				</div>
			</div>
		{/each}
	</div>

	{#if error}
		<div class="p-2 text-center text-sm text-red-500">
			{error}
		</div>
	{/if}

	<div class="border-t border-gray-200 p-4">
		<form on:submit|preventDefault={sendMessage} class="flex gap-2">
			<input
				type="text"
				bind:value={messageInput}
				placeholder={isLocked && room.lockedBy !== userId ? 'Room is locked' : 'Type a message...'}
				disabled={!canSendMessage()}
				class="flex-1 rounded-md border border-gray-300 px-4 py-2 focus:border-indigo-500 focus:outline-none disabled:bg-gray-100 disabled:opacity-50"
			/>
			<button
				type="submit"
				disabled={!canSendMessage()}
				class="rounded-md bg-indigo-600 px-4 py-2 text-white hover:bg-indigo-700 disabled:opacity-50"
			>
				Send
			</button>
		</form>
	</div>
</div>
