import { NextRequest, NextResponse } from "next/server";

export async function GET(req: NextRequest) {
    const token = req.headers.get('Authorization')?.split(' ')[1];

    const response = await fetch(`${process.env.BACKEND_ROOT_URL}/api/v1/rooms`, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
        },
    });

    if (response.status === 401) {
        return NextResponse.json({ error: "Unauthorized", shouldSignOut: true }, { status: 401 });
    }

    const data = await response.json();

    return NextResponse.json(data.rooms);
}

export async function POST(req: NextRequest) {
    const { room_id, user_id, nickname } = await req.json()
    const token = req.headers.get('Authorization')?.split(' ')[1];

    const response = await fetch(`${process.env.BACKEND_ROOT_URL}/api/v1/rooms/${room_id}/register-user`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
            user_id: user_id,
            nickname: nickname,
        }),
    });

    if (response.status === 401) {
        return NextResponse.json({ error: "Unauthorized", shouldSignOut: true }, { status: 401 });
    }

    const data = await response.json();

    return NextResponse.json({
        user_id: data.users[0].id,
        room_id: data.id,
        nickname: data.users[0].nickname,
    });
} 