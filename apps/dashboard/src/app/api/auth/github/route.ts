import { NextResponse } from 'next/server';

export async function GET(request: Request) {
  let clientId = '';

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
    const appUrl = process.env.TRIAGE_DASHBOARD_URL || 'http://localhost:3000';
    return NextResponse.redirect(`${appUrl}?user=algotyrnt&auth=dev`);
  }

  // Use the dashboard URL as the base for the callback
  const baseUrl = process.env.TRIAGE_DASHBOARD_URL || 'http://localhost:3000';
  const callbackUrl = `${baseUrl}/api/auth/github/callback`;

  // Note: if the client ID belongs to a GitHub App, scopes are ignored.
  // We only request user identity scopes.
  const redirectUri = `https://github.com/login/oauth/authorize?client_id=${clientId}&redirect_uri=${encodeURIComponent(callbackUrl)}&scope=user:email,read:user`;

  return NextResponse.redirect(redirectUri);
}
