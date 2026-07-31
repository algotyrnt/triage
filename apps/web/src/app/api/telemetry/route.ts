/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import { NextResponse } from "next/server";

const ENGINE_URL = process.env.ENGINE_URL || "http://localhost:8080/api/v1/telemetry";

export async function POST(request: Request) {
  try {
    const body = await request.json();

    const engineResponse = await fetch(ENGINE_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
    });

    const data = await engineResponse.json();
    return NextResponse.json(data, { status: engineResponse.status });
  } catch (error) {
    return NextResponse.json(
      {
        status: "error",
        error: error instanceof Error ? error.message : "Engine communication error",
      },
      { status: 500 }
    );
  }
}
