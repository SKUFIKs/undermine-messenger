import React from 'react';
import type { Message } from '../types';

interface MessageBubbleProps {
  message: Message;
}

export const MessageBubble: React.FC<MessageBubbleProps> = ({ message }) => {
  const formatTime = (dateStr: string) => {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return '';
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const getStatusIcon = (status: Message['status']) => {
    switch (status) {
      case 'sending':
        return ' 🕒';
      case 'sent':
        return ' ✓';
      case 'delivered':
      case 'read':
        return ' ✓✓';
      case 'failed':
        return ' ⚠️';
      default:
        return '';
    }
  };

  return (
    <div
      className={`flex w-full mb-2 ${
        message.isOutgoing ? 'justify-end' : 'justify-start'
      }`}
    >
      <div
        className={`max-w-[75%] rounded-[10px] px-3.5 py-2 relative text-[13px] leading-relaxed shadow-sm break-words select-text ${
          message.isOutgoing
            ? 'bg-[#C76029] text-white'
            : 'bg-[#2A2A2A] text-white'
        }`}
      >
        <div className="whitespace-pre-wrap">{message.content}</div>

        <div className="flex items-center justify-end gap-1 mt-1 text-[10px] text-[#E0E0E0]/80 select-none">
          <span>{formatTime(message.createdAt)}</span>
          {message.isOutgoing && <span>{getStatusIcon(message.status)}</span>}
        </div>
      </div>
    </div>
  );
};
