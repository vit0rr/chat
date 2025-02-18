<script lang="ts">
	import type { Room } from '$lib/types/chat';
	import { onMount, onDestroy } from 'svelte';
	import { fetchMessages } from '$lib/utils/chat';
	import { auth } from '$lib/stores/auth';
	import Lock from '$lib/components/icons/Lock.svelte';

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

<div class="flex h-[600px] flex-col rounded-2xl bg-white shadow-lg">
	<!-- Header -->
	<div class="flex items-center justify-between border-b border-gray-200 px-6 py-4">
		<div class="flex items-center space-x-3">
			<div
				class="flex h-10 w-10 items-center justify-center rounded-full bg-indigo-100 text-indigo-600"
			>
				<span class="text-lg font-semibold">{room.id?.slice(0, 2).toUpperCase()}</span>
			</div>
			<div>
				<h2 class="text-lg font-semibold text-gray-900">Room #{room.id?.slice(0, 8)}</h2>
				<div class="flex items-center gap-2 text-sm text-gray-500">
					<span>{connected ? 'Connected' : 'Disconnected'}</span>
					<span>•</span>
					<span>as {nickname}</span>
				</div>
			</div>
		</div>
		<button
			on:click={toggleRoomLock}
			disabled={!connected}
			class="rounded-xl px-4 py-2 text-sm font-medium transition-colors {isLocked &&
			room.lockedBy === userId
				? 'bg-red-50 text-red-700 hover:bg-red-100'
				: 'bg-indigo-50 text-indigo-700 hover:bg-indigo-100'} disabled:opacity-50"
		>
			{#if isLocked}
				{#if room.lockedBy === userId}
					<span class="flex items-center gap-2">
						<Lock />
						Unlock Room
					</span>
				{:else}
					<span class="flex items-center gap-2">
						<Lock />
						Locked
					</span>
				{/if}
			{:else}
				<span class="flex items-center gap-2">
					<Lock />
					Lock Room
				</span>
			{/if}
		</button>
	</div>

	<!-- Messages -->
	<div
		bind:this={messageContainer}
		on:scroll={handleScroll}
		class="flex-1 space-y-4 overflow-y-auto px-6 py-4"
	>
		{#if loading && currentPage === 1}
			<div class="flex justify-center">
				<div class="rounded-lg bg-gray-50 px-4 py-2 text-sm text-gray-500">Loading messages...</div>
			</div>
		{/if}

		{#if hasMore}
			<div class="flex justify-center">
				<div class="rounded-lg bg-gray-50 px-4 py-2 text-xs text-gray-500">
					{loading ? 'Loading more messages...' : 'Scroll up to load more'}
				</div>
			</div>
		{/if}

		{#each messages as message}
			<div
				class="flex {message.sender_id === userId || message.fromUserId === userId
					? 'justify-end'
					: 'justify-start'}"
			>
				<div
					class="max-w-[70%] space-y-1 {message.sender_id === userId ||
					message.fromUserId === userId
						? 'items-end'
						: 'items-start'}"
				>
					{#if message.type !== 'system'}
						<span class="text-xs font-medium text-gray-500">{message.nickname}</span>
					{/if}
					<div
						class="rounded-2xl px-4 py-2 {message.type === 'system'
							? 'bg-gray-100 text-sm italic text-gray-600'
							: message.sender_id === userId || message.fromUserId === userId
								? 'bg-indigo-600 text-white'
								: 'bg-gray-100 text-gray-900'}"
					>
						<p>{message.content || message.message}</p>
					</div>
					{#if message.type !== 'system'}
						<span class="text-xs text-gray-400">
							{new Date(message.timestamp || message.createdAt).toLocaleTimeString([], {
								hour: '2-digit',
								minute: '2-digit'
							})}
						</span>
					{/if}
				</div>
			</div>
		{/each}
	</div>

	{#if error}
		<div class="border-t border-red-100 bg-red-50 px-6 py-3 text-center text-sm text-red-600">
			{error}
		</div>
	{/if}

	<!-- Input -->
	<div class="border-t border-gray-200 px-6 py-4">
		<form on:submit|preventDefault={sendMessage} class="flex space-x-4">
			<input
				type="text"
				bind:value={messageInput}
				placeholder={isLocked && room.lockedBy !== userId ? 'Room is locked' : 'Type a message...'}
				disabled={!canSendMessage()}
				class="flex-1 rounded-xl border-0 bg-gray-50 px-4 py-3 text-gray-900 placeholder:text-gray-400 focus:bg-gray-100 focus:outline-none focus:ring-0 disabled:bg-gray-100 disabled:opacity-50"
			/>
			<button
				type="submit"
				disabled={!canSendMessage()}
				class="rounded-xl bg-indigo-600 px-6 py-3 text-sm font-semibold text-white transition-colors hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-600 focus:ring-offset-2 disabled:opacity-50"
			>
				Send
			</button>
		</form>
	</div>
</div>
