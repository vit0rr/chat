"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { Room } from "@/lib/rooms";
import Link from "next/link";
import Header from "@/components/Header";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Loader2 } from "lucide-react";

export default function RoomsPage() {
  const { token } = useAuth();
  const [rooms, setRooms] = useState<Room[]>([]);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    const fetchRooms = async () => {
      if (!token) return;

      try {
        setLoading(true);
        const roomsResponse = await fetch(`/api/rooms`, {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });
        const roomsData = await roomsResponse.json();
        setRooms(roomsData);
      } catch (error) {
        console.error("Error fetching rooms:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchRooms();
  }, [token]);

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <Header />

      <main className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-8">
          <h1 className="text-3xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-500 to-blue-600">
            Available Rooms
          </h1>
          <Button
            className="bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white shadow-lg hover:shadow-xl transition-all"
            onClick={() => router.push("/rooms/create")}
          >
            Create Room
          </Button>
        </div>

        {loading ? (
          <div className="flex justify-center items-center h-64">
            <Loader2 className="h-8 w-8 animate-spin" />
          </div>
        ) : rooms.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center space-y-4 pt-8 pb-8">
              <CardTitle>No rooms found</CardTitle>
              <CardDescription>
                You haven&apos;t joined any rooms yet.
              </CardDescription>
              <Button onClick={() => router.push("/rooms/create")}>
                Create Your First Room
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {rooms.map((room) => (
              <Card
                key={room.room_id}
                className="hover:shadow-lg transition-shadow"
              >
                <Link href={`/rooms/${room.room_id}`}>
                  <CardHeader>
                    <div className="flex justify-between items-center">
                      <CardTitle className="text-xl">
                        Room {room.room_id.substring(0, 8)}
                      </CardTitle>
                      {room.locked_by && (
                        <Badge variant="secondary">Locked</Badge>
                      )}
                    </div>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4">
                      <div>
                        <h3 className="text-sm text-muted-foreground mb-2">
                          Members:
                        </h3>
                        <div className="flex flex-wrap gap-2">
                          {room.users.map((user) => (
                            <Badge key={user.id} variant="outline">
                              {user.nickname}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    </div>
                  </CardContent>
                  <CardFooter>
                    <p className="text-xs text-muted-foreground">
                      Created: {new Date(room.created_at).toLocaleDateString()}
                    </p>
                  </CardFooter>
                </Link>
              </Card>
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
