import { NextResponse } from 'next/server';

function getBaseUrl(request: Request): string {
  const host = request.headers.get('x-forwarded-host') || request.headers.get('host');
  const proto = request.headers.get('x-forwarded-proto') || 'http';
  if (host) {
    return `${proto}://${host}`;
  }
  return new URL(request.url).origin;
}

export async function GET(request: Request) {
  let clientId = '';
  const baseUrl = getBaseUrl(request);

  try {
    const engineUrl = process.env.TRIAGE_ENGINE_URL || 'http://localhost:8080';
    const res = await fetch(`${engineUrl}/api/v1/setup/oauth`);
    if (res.ok) {
      const data = await res.json();
      clientId = data.client_id;
    }
  } catch (e) {
    console.error('Failed to fetch oauth config from engine', e);
  }

  if (!clientId) {
    return NextResponse.redirect(`${baseUrl}?user=algotyrnt&auth=dev`);
  }

  const callbackUrl = `${baseUrl}/api/auth/github/callback`;

  // Note: if the client ID belongs to a GitHub App, scopes are ignored.
  // We only request user identity scopes.
  const redirectUri = `https://github.com/login/oauth/authorize?client_id=${clientId}&redirect_uri=${encodeURIComponent(callbackUrl)}&scope=user:email,read:user`;

  return NextResponse.redirect(redirectUri);
}
