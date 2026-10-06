import React from 'react';

// Palette matching Go's internal/ui/components/avatar.go
const AVATAR_PALETTE = [
  '#C76029',
  '#3E7EB5',
  '#3B976C',
  '#8E549F',
  '#D34747',
  '#C88820',
];

interface AvatarProps {
  title: string;
  size?: number; // pixels, default 44
  className?: string;
}

export const Avatar: React.FC<AvatarProps> = ({ title, size = 44, className = '' }) => {
  const getInitials = (name: string): string => {
    const parts = name.trim().split(/\s+/).filter(Boolean);
    if (parts.length === 0) return '?';
    if (parts.length === 1) {
      return parts[0].slice(0, 2).toUpperCase();
    }
    return (parts[0][0] + parts[1][0]).toUpperCase();
  };

  const pickColor = (name: string): string => {
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = (hash * 31 + name.charCodeAt(i)) >>> 0;
    }
    return AVATAR_PALETTE[hash % AVATAR_PALETTE.length];
  };

  const bgColor = pickColor(title || 'U');
  const initials = getInitials(title || 'User');
  const fontSize = Math.floor(size * 0.42);

  return (
    <div
      className={`rounded-full flex items-center justify-center font-bold text-white shrink-0 select-none ${className}`}
      style={{
        width: `${size}px`,
        height: `${size}px`,
        backgroundColor: bgColor,
        fontSize: `${fontSize}px`,
      }}
    >
      {initials}
    </div>
  );
};
