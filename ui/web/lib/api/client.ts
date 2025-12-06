// For Server Components, we can directly call the Go server or use the Next.js API route
// Using the Next.js API route is better for consistency and CORS handling
export async function fetchMetrics(): Promise<string> {
  // In Server Components, we need an absolute URL
  // Use the Next.js API route which proxies to the Go server
  const baseUrl = process.env.NEXT_PUBLIC_APP_URL || 
    (process.env.VERCEL_URL ? `https://${process.env.VERCEL_URL}` : 'http://localhost:3000');
  
  const response = await fetch(`${baseUrl}/api/metrics`, {
    next: { revalidate: 5 }, // Revalidate every 5 seconds for auto-refresh
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch metrics: ${response.statusText}`);
  }

  return response.text();
}

export async function fetchHealthStats() {
  const baseUrl = process.env.NEXT_PUBLIC_APP_URL || 
    (process.env.VERCEL_URL ? `https://${process.env.VERCEL_URL}` : 'http://localhost:3000');
  
  const response = await fetch(`${baseUrl}/api/balto/health/stats`, {
    next: { revalidate: 5 },
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch health stats: ${response.statusText}`);
  }

  return response.json();
}

export async function fetchServices() {
  const baseUrl = process.env.NEXT_PUBLIC_APP_URL || 
    (process.env.VERCEL_URL ? `https://${process.env.VERCEL_URL}` : 'http://localhost:3000');
  
  const response = await fetch(`${baseUrl}/api/balto/services`, {
    next: { revalidate: 10 },
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch services: ${response.statusText}`);
  }

  return response.json();
}

