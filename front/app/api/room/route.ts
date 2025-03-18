import { NextRequest, NextResponse } from "next/server";

export async function GET(req: NextRequest) {
    const roomId = req.nextUrl.searchParams.get('roomId');
    const token = req.headers.get('Authorization')?.split(' ')[1];

    const response = await fetch(`${process.env.BACKEND_ROOT_URL}/api/v1/rooms/${roomId}`, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
            'X-API-Key': process.env.API_KEY || '',
        },
    });

    if (response.status === 401) {
        return NextResponse.json({ error: "Unauthorized", shouldSignOut: true }, { status: 401 });
    }

    const data = await response.json();

    return NextResponse.json(data);
}

export async function POST(req: NextRequest) {
    const roomId = req.nextUrl.searchParams.get('roomId');
    const userId = req.nextUrl.searchParams.get('userId');
    const nickname = req.nextUrl.searchParams.get('nickname');

    const token = req.headers.get('Authorization')?.split(' ')[1];

    const response = await fetch(`${process.env.BACKEND_ROOT_URL}/api/v1/rooms/${roomId}/register-user`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
            'X-API-Key': process.env.API_KEY || '',
        },
        body: JSON.stringify({
            user_id: userId,
            nickname: nickname,
        }),
    });

    if (response.status === 401) {
        return NextResponse.json({ error: "Unauthorized", shouldSignOut: true }, { status: 401 });
    }

    const data = await response.json();

    return NextResponse.json(data);
} 