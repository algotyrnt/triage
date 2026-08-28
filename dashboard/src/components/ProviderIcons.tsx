/**
 * Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import React from 'react';

interface IconProps {
  className?: string;
  size?: number;
}

/**
 * Google Gemini Official Sparkle Logo
 */
export const GeminiIcon: React.FC<IconProps> = ({
  className = 'w-4 h-4 text-indigo-500',
  size,
}) => (
  <svg
    viewBox="0 0 24 24"
    fill="currentColor"
    width={size}
    height={size}
    className={className}
    aria-label="Google Gemini"
  >
    <path d="M12 24C12 17.3726 6.62742 12 0 12C6.62742 12 12 6.62742 12 0C12 6.62742 17.3726 12 24 12C17.3726 12 12 17.3726 12 24Z" />
  </svg>
);

/**
 * OpenAI Official Aperture Logo
 */
export const OpenAIIcon: React.FC<IconProps> = ({
  className = 'w-4 h-4 text-emerald-600',
  size,
}) => (
  <svg
    viewBox="0 0 24 24"
    fill="currentColor"
    width={size}
    height={size}
    className={className}
    aria-label="OpenAI"
  >
    <path d="M22.2819 9.8211a5.9847 5.9847 0 0 0-.5157-4.9108 6.0462 6.0462 0 0 0-6.5098-2.9A6.0651 6.0651 0 0 0 4.9807 4.1818a5.9847 5.9847 0 0 0-3.9977 2.9 6.0462 6.0462 0 0 0 .7427 7.0966 5.98 5.98 0 0 0 .511 4.9107 6.051 6.051 0 0 0 6.5146 2.9001A5.9847 5.9847 0 0 0 13.2599 24a6.0557 6.0557 0 0 0 5.7718-4.2058 5.9894 5.9894 0 0 0 3.9977-2.9001 6.0557 6.0557 0 0 0-.7475-7.0729zm-9.022 12.6081a4.4755 4.4755 0 0 1-2.8764-1.0408l.1419-.0804 4.7783-2.7582a.7948.7948 0 0 0 .3927-.6813v-6.7369l2.02 1.1683a.071.071 0 0 1 .038.052v5.5826a4.5045 4.5045 0 0 1-4.4945 4.4947zm-9.6607-4.1254a4.4708 4.4708 0 0 1-.5346-3.0137l.142.0852 4.783 2.7582a.7712.7712 0 0 0 .7806 0l5.8428-3.3685v2.3324a.0804.0804 0 0 1-.0332.0615L9.74 19.9502a4.4992 4.4992 0 0 1-6.1408-1.6464zM2.3408 7.8956a4.485 4.485 0 0 1 2.3655-1.9728V11.6a.7664.7664 0 0 0 .3879.6765l5.8144 3.3543-2.0201 1.1683a.0757.0757 0 0 1-.071 0l-4.8303-2.7866A4.504 4.504 0 0 1 2.3408 7.8956zm16.0993 3.8558L12.5973 8.3829l2.02-1.1635a.0804.0804 0 0 1 .071 0l4.8303 2.7913a4.4944 4.4944 0 0 1-.6765 8.1042v-5.6772a.79.79 0 0 0-.402-.6863zm2.0107-3.0231l-.142-.0852-4.7735-2.7818a.7759.7759 0 0 0-.7854 0L9.409 9.2297V6.8974a.0662.0662 0 0 1 .0284-.0615l4.8303-2.7866a4.4992 4.4992 0 0 1 6.6802 4.66zM8.3065 12.863l-2.02-1.1635a.0804.0804 0 0 1-.038-.0567V6.0748a4.4992 4.4992 0 0 1 7.3757-3.4537l-.142.0805L8.704 5.4598a.7948.7948 0 0 0-.3927.6813v6.7219zm1.093-2.0624l2.6-1.5 2.6 1.5v3l-2.6 1.5-2.6-1.5z" />
  </svg>
);

/**
 * Anthropic Claude Official Monogram Logo
 */
export const AnthropicIcon: React.FC<IconProps> = ({
  className = 'w-4 h-4 text-amber-600',
  size,
}) => (
  <svg
    viewBox="0 0 24 24"
    fill="currentColor"
    width={size}
    height={size}
    className={className}
    aria-label="Anthropic Claude"
  >
    <path d="M4.5 3h4.2l6.8 18h-4.2L4.5 3zm10.8 0H19.5l-4.2 11.1h-4.2L15.3 3z" />
  </svg>
);

/**
 * Ollama Official Llama Silhouette Logo
 */
export const OllamaIcon: React.FC<IconProps> = ({
  className = 'w-4 h-4 text-purple-600',
  size,
}) => (
  <svg
    viewBox="0 0 24 24"
    fill="currentColor"
    width={size}
    height={size}
    className={className}
    aria-label="Ollama"
  >
    <path d="M10 2a1 1 0 0 0-1 1v2H8a1 1 0 0 0-1 1v4.5l-1.5 1.5A3 3 0 0 0 5 14v2a3 3 0 0 0 3 3h1v2a1 1 0 0 0 2 0v-2h2v2a1 1 0 0 0 2 0v-2h1a3 3 0 0 0 3-3v-2a3 3 0 0 0-.5-1.5L17 10.5V6a1 1 0 0 0-1-1h-1V3a1 1 0 0 0-1-1h-4zm-1 8a1.25 1.25 0 1 1 0-2.5 1.25 1.25 0 0 1 0 2.5zm6 0a1.25 1.25 0 1 1 0-2.5 1.25 1.25 0 0 1 0 2.5z" />
  </svg>
);

/**
 * Helper to get the correct icon for any provider ID
 */
export const ProviderIcon: React.FC<{ provider?: string; className?: string; size?: number }> = ({
  provider,
  className,
  size,
}) => {
  switch (provider?.toLowerCase()) {
    case 'gemini':
      return <GeminiIcon className={className || 'w-4 h-4 text-indigo-500'} size={size} />;
    case 'openai':
      return <OpenAIIcon className={className || 'w-4 h-4 text-emerald-600'} size={size} />;
    case 'anthropic':
      return <AnthropicIcon className={className || 'w-4 h-4 text-amber-600'} size={size} />;
    case 'ollama':
    case 'custom':
      return <OllamaIcon className={className || 'w-4 h-4 text-purple-600'} size={size} />;
    default:
      return <GeminiIcon className={className || 'w-4 h-4 text-indigo-500'} size={size} />;
  }
};
