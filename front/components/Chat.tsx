import { useState, useRef, useEffect } from "react";
import { useWebSocket } from "@/lib/useWebSocket";
import { Loader2 } from "lucide-react";

type ChatProps = {
  roomId: string;
  userId: string;
  nickname: string;
  token: string;
};

export default function Chat({ roomId, userId, nickname, token }: ChatProps) {
  const {
    messages,
    sendMessage,
    isConnected,
    error,
    isLoadingHistory,
    hasMore,
    loadMoreMessages,
  } = useWebSocket(roomId, userId, nickname, token);
  const [newMessage, setNewMessage] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const chatContainerRef = useRef<HTMLDivElement>(null);

  const handleScroll = () => {
    const container = chatContainerRef.current;
    if (
      container &&
      container.scrollTop === 0 &&
      !isLoadingHistory &&
      hasMore
    ) {
      loadMoreMessages();
    }
  };

  useEffect(() => {
    if (!isLoadingHistory) {
      scrollToBottom();
    }
  }, [messages, isLoadingHistory]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newMessage.trim()) return;

    sendMessage(newMessage);
    setNewMessage("");
  };

  if (error) {
    return (
      <div className="bg-red-50 dark:bg-red-900/20 p-4 rounded-lg text-center">
        <p className="text-red-600 dark:text-red-400">{error}</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-[600px]">
      <div
        ref={chatContainerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto mb-4 space-y-4 p-4 dark:bg-gray-700/50 rounded-lg"
      >
        {isLoadingHistory && (
          <div className="text-center py-2">
            <Loader2 className="w-10 h-10 animate-spin" />
          </div>
        )}

        {hasMore && !isLoadingHistory && messages.length > 0 && (
          <button
            onClick={loadMoreMessages}
            className="w-full text-blue-500 hover:text-blue-600 text-sm py-2"
          >
            Load more messages
          </button>
        )}

        {!isLoadingHistory && messages.length === 0 && (
          <div className="text-center py-8">
            <p className="text-gray-500 dark:text-gray-400">
              No messages yet. Be the first to send a message!
            </p>
          </div>
        )}

        {messages.map((message, index) => (
          <div
            key={`${message.timestamp}-${index}`}
            className={`flex ${
              message.sender_id === userId ? "justify-end" : "justify-start"
            }`}
          >
            <div
              className={`max-w-[70%] rounded-lg p-3 ${
                message.sender_id === userId
                  ? "bg-blue-500 text-white"
                  : "bg-gray-200 dark:bg-gray-600"
              }`}
            >
              {message.type === "system" ? (
                <p className="text-sm italic text-gray-500 dark:text-gray-400">
                  {message.content}
                </p>
              ) : (
                <>
                  <p className="text-xs mb-1 opacity-75">{message.nickname}</p>
                  <p>{message.content}</p>
                  <p className="text-xs mt-1 opacity-75">
                    {new Date(message.timestamp).toLocaleTimeString()}
                  </p>
                </>
              )}
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      <form onSubmit={handleSubmit} className="flex gap-2">
        <input
          type="text"
          value={newMessage}
          onChange={(e) => setNewMessage(e.target.value)}
          placeholder={isConnected ? "Type a message..." : "Connecting..."}
          disabled={!isConnected}
          className="flex-1 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 p-2 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:text-white"
        />
        <button
          type="submit"
          disabled={!isConnected || !newMessage.trim()}
          className="bg-blue-500 text-white px-4 py-2 rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          Send
        </button>
      </form>
    </div>
  );
}
