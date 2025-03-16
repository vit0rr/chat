import { NextRequest, NextResponse } from 'next/server';

export async function POST(req: NextRequest) {
    const { email, password, nickname } = await req.json();

    const response = await fetch(`${process.env.BACKEND_ROOT_URL}/api/v1/auth/register`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email, password, nickname }),
    });

    const data = await response.json();

    return NextResponse.json(data);
} 