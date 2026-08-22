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
  const url = new URL(request.url);
  const code = url.searchParams.get('code');
  const baseUrl = getBaseUrl(request);

  if (!code) {
    return NextResponse.redirect(`${baseUrl}?auth=error&reason=missing_code`);
  }

  let clientId = '';
  let clientSecret = '';

  try {
    const engineUrl = process.env.TRIAGE_ENGINE_URL || 'http://localhost:8080';
    const res = await fetch(`${engineUrl}/api/v1/setup/oauth`);
    if (res.ok) {
      const data = await res.json();
      clientId = data.client_id;
      clientSecret = data.client_secret;
    }
  } catch (e) {
    console.error('Failed to fetch oauth config from engine', e);
  }

  if (!clientId || !clientSecret) {
    return NextResponse.redirect(`${baseUrl}?auth=error&reason=oauth_not_configured`);
  }

  try {
    // Exchange the authorization code for an access token
    const tokenRes = await fetch('https://github.com/login/oauth/access_token', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: JSON.stringify({
        client_id: clientId,
        client_secret: clientSecret,
        code,
      }),
    });

    if (!tokenRes.ok) {
      console.error('Failed to exchange token:', await tokenRes.text());
      return NextResponse.redirect(`${baseUrl}?auth=error&reason=token_exchange_failed`);
    }

    const tokenData = await tokenRes.json();
    const accessToken = tokenData.access_token;

    if (!accessToken) {
      console.error('No access token in response:', tokenData);
      return NextResponse.redirect(`${baseUrl}?auth=error&reason=no_access_token`);
    }

    // Pass the raw GitHub token back to the dashboard UI for it to verify directly
    return NextResponse.redirect(
      `${baseUrl}?token=${encodeURIComponent(accessToken)}&auth=success`,
    );
  } catch (err) {
    console.error('Callback processing error:', err);
    return NextResponse.redirect(`${baseUrl}?auth=error&reason=internal_error`);
  }
}
