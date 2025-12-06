import { NextRequest, NextResponse } from 'next/server';

const API_BASE = process.env.METRICS_URL || 'http://localhost:5500';

export const dynamic = 'force-dynamic';

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const window = searchParams.get('window') || '60';

  try {
    const response = await fetch(`${API_BASE}/balto/metrics/aggregated?window=${window}`, {
      cache: 'no-store',
    });

    if (!response.ok) {
      console.error('Backend error:', response.status, response.statusText);
      return NextResponse.json(
        { success: false, error: `Backend error: ${response.statusText}`, data: null },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error fetching aggregated metrics:', error);
    return NextResponse.json(
      { success: false, error: 'Failed to fetch metrics', data: null },
      { status: 500 }
    );
  }
}
