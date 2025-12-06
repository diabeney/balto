import { NextRequest, NextResponse } from 'next/server';

const API_BASE = process.env.METRICS_URL || 'http://localhost:5500';

export const dynamic = 'force-dynamic';

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const window = searchParams.get('window') || '60';

  try {
    const response = await fetch(`${API_BASE}/balto/metrics/snapshot?window=${window}`, {
      cache: 'no-store',
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: `Failed to fetch snapshot: ${response.statusText}` },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error fetching snapshot:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

