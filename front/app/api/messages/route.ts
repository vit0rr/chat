import { NextRequest, NextResponse } from "next/server";

export async function GET(req: NextRequest) {
    const roomId = req.nextUrl.searchParams.get('roomId');
    const page = req.nextUrl.searchParams.get('page') || '1';
    const limit = req.nextUrl.searchParams.get('limit') || '50';
    const token = req.headers.get('Authorization')?.split(' ')[1];

    if (!token) {
        return NextResponse.json({ error: 'Authentication token is required' }, { status: 401 });
    }

    try {
        const response = await fetch(
            `${process.env.BACKEND_ROOT_URL}/api/v1/rooms/${roomId}/messages?page=${page}&limit=${limit}`,
            {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`,
                    'X-API-Key': process.env.API_KEY || '',
                },
            }
        );

        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        console.error('Error fetching messages:', error);
        return NextResponse.json({ error: 'Failed to fetch messages' }, { status: 500 });
    }
} 