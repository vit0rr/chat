"use client";

import { useEffect, useState, use } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { getRoom, Room, joinRoom } from "@/lib/rooms";
import Link from "next/link";
import Header from "@/components/Header";
import Chat from "@/components/Chat";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Separator } from "@/components/ui/separator";
import { ChevronLeft, Loader2, LockIcon } from "lucide-react";

export default function RoomDetailPage({
  params,
}: {
  params: Promise<{ roomId: string }>;
}) {
  const resolvedParams = use(params);
  const roomId = resolvedParams.roomId;

  const { user, token, isAuthenticated } = useAuth();
  const [room, setRoom] = useState<Room | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [isJoining, setIsJoining] = useState(false);
  const router = useRouter();

  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login");
      return;
    }

    const fetchRoom = async () => {
      if (!token || !roomId) return;

      try {
        setLoading(true);
        setError("");
        const roomData = await getRoom(roomId, token);
        setRoom(roomData);
      } catch (err: any) {
        console.error("Error fetching room:", err);
        setError(err.response?.data?.message || "Failed to load room");
        if (err.response?.status === 404) {
          router.push("/rooms");
        }
      } finally {
        setLoading(false);
      }
    };

    fetchRoom();
  }, [isAuthenticated, token, roomId, router]);

  const isUserInRoom = room?.users?.some(
    (roomUser) => roomUser.id === user?.id
  );

  console.log({
    roomUser: room?.users,
    user: user,
  });

  console.log("isUserInRoom", isUserInRoom);

  const handleJoinRoom = async () => {
    if (!user || !token) {
      router.push("/login");
      return;
    }

    try {
      setIsJoining(true);
      setError("");
      await joinRoom(roomId, user.id, user.nickname, token);
      // Fetch the updated room data
      const updatedRoom = await getRoom(roomId, token);
      setRoom(updatedRoom);
    } catch (err) {
      console.error("Error joining room:", err);
      setError(err instanceof Error ? err.message : "Failed to join room");
    } finally {
      setIsJoining(false);
    }
  };

  if (!isAuthenticated) {
    return null;
  }

  if (loading) {
    return (
      <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
        <Header />
        <main className="flex-1 container mx-auto px-4 py-8">
          <div className="flex justify-center items-center h-64">
            <Loader2 className="h-8 w-8 animate-spin" />
          </div>
        </main>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
        <Header />
        <main className="flex-1 container mx-auto px-4 py-8">
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
          <Button variant="link" asChild className="mt-4">
            <Link href="/rooms">Back to Rooms</Link>
          </Button>
        </main>
      </div>
    );
  }

  if (!room) {
    return (
      <div className="container mx-auto px-4 py-8">
        <Card>
          <CardContent className="flex flex-col items-center justify-center space-y-4 pt-8 pb-8">
            <CardTitle>Room not found</CardTitle>
            <p className="text-muted-foreground">
              The room you're looking for doesn't exist or you don't have
              access.
            </p>
            <Button asChild>
              <Link href="/rooms">Back to Rooms</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
      <Header />
      <main className="flex-1 container mx-auto px-4 py-8">
        <div className="space-y-6 max-w-5xl mx-auto">
          <Button variant="ghost" asChild className="gap-2">
            <Link href="/rooms">
              <ChevronLeft className="h-4 w-4" />
              Back to Rooms
            </Link>
          </Button>

          <div className="grid grid-cols-1 md:grid-cols-12 gap-6">
            {/* Sidebar with room info and members */}
            <Card className="md:col-span-4">
              <CardHeader>
                <div className="flex justify-between items-center">
                  <CardTitle>Room {roomId.substring(0, 8)}</CardTitle>
                  {room?.locked_by && room.users && (
                    <Badge variant="secondary" className="gap-1">
                      <LockIcon className="h-3 w-3" />
                      Locked
                    </Badge>
                  )}
                </div>
              </CardHeader>

              <CardContent className="space-y-6">
                {!isUserInRoom && (
                  <Button
                    onClick={handleJoinRoom}
                    disabled={isJoining}
                    className="w-full"
                  >
                    {isJoining ? (
                      <>
                        <Loader2 className="animate-spin mr-2 h-4 w-4" />
                        Joining...
                      </>
                    ) : (
                      "Join Room"
                    )}
                  </Button>
                )}

                <div>
                  <h2 className="text-sm font-semibold mb-3">Members</h2>
                  <div className="space-y-2">
                    {room?.users?.map((roomUser) => {
                      if (!roomUser?.nickname) return null;

                      return (
                        <div
                          key={roomUser.id}
                          className="flex items-center gap-2 p-2 bg-muted rounded-lg"
                        >
                          <Avatar className="h-6 w-6">
                            <AvatarFallback className="text-xs">
                              {roomUser.nickname.charAt(0).toUpperCase()}
                            </AvatarFallback>
                          </Avatar>
                          <div>
                            <p className="text-sm font-medium">
                              {roomUser.nickname}
                            </p>
                            {user && roomUser.id === user.id && (
                              <span className="text-xs text-muted-foreground">
                                You
                              </span>
                            )}
                          </div>
                          {room.locked_by === roomUser.id && (
                            <LockIcon className="h-3 w-3 ml-auto text-muted-foreground" />
                          )}
                        </div>
                      );
                    })}
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Chat section */}
            <Card className="md:col-span-8">
              <CardHeader>
                <CardTitle className="text-lg">Chat</CardTitle>
              </CardHeader>
              <CardContent>
                {error && (
                  <Alert variant="destructive" className="mb-4">
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}

                {isUserInRoom ? (
                  <div className="h-[600px]">
                    <Chat
                      roomId={roomId}
                      userId={user?.id || ""}
                      nickname={user?.nickname || ""}
                      token={token || ""}
                    />
                  </div>
                ) : (
                  <div className="flex items-center justify-center h-[600px] bg-muted/50 rounded-lg">
                    <p className="text-muted-foreground">
                      Join this room to participate in the chat
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </main>
    </div>
  );
}
