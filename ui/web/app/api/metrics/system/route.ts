import { NextRequest, NextResponse } from 'next/server';

const API_BASE = process.env.METRICS_URL || 'http://localhost:5500';

export const dynamic = 'force-dynamic';

export async function GET(request: NextRequest) {
  try {
    const response = await fetch(`${API_BASE}/balto/metrics/system`, {
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
    console.error('Error fetching system metrics:', error);
    return NextResponse.json(
      { success: false, error: 'Failed to fetch system metrics', data: null },
      { status: 500 }
    );
  }
}

