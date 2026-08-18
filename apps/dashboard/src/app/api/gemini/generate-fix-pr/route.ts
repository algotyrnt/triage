import { NextRequest, NextResponse } from 'next/server';

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const engineUrl = process.env.NEXT_PUBLIC_ENGINE_URL || 'http://localhost:8080';
    const res = await fetch(`${engineUrl}/api/v1/gemini/generate-patch`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (error: any) {
    return NextResponse.json(
      { error: error?.message || 'Internal server error' },
      { status: 500 },
    );
  }
}
