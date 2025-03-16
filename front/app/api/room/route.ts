import { NextRequest, NextResponse } from "next/server";

export async function GET(req: NextRequest) {
    const roomId = req.nextUrl.searchParams.get('roomId');
    const token = req.headers.get('Authorization')?.split(' ')[1];

    const response = await fetch(`${process.env.BACKEND_ROOT_URL}/api/v1/rooms/${roomId}`, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
        },
    });

    const data = await response.json();

    return NextResponse.json(data);
} 