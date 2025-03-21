import { NextRequest, NextResponse } from 'next/server';

export async function POST(req: NextRequest) {
    const { email, password } = await req.json();

    const response = await fetch(`${process.env.BACKEND_ROOT_URL}/api/v1/auth/login`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${process.env.API_KEY}`,
        },
        body: JSON.stringify({ email, password }),
    });

    const data = await response.json();

    if (!response.ok) return NextResponse.json({ error: data.error }, { status: data.code });

    return NextResponse.json(data);
} 