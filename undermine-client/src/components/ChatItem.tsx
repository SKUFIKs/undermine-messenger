import React from 'react';
import type { Chat } from '../types';
import { Avatar } from './Avatar';
import { Users } from 'lucide-react';

interface ChatItemProps {
  chat: Chat;
  isSelected: boolean;
  onSelect: (chat: Chat) => void;
  isGroup?: boolean;
  memberCount?: number;
}

export const ChatItem: React.FC<ChatItemProps> = ({ chat, isSelected, onSelect, isGroup, memberCount }) => {
  const formatTime = (dateStr: string) => {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return '';
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const preview = chat.lastMessage?.content || 'Нет сообщений';
  const truncatedPreview = preview.length > 36 ? preview.slice(0, 33) + '...' : preview;

  return (
    <div
      onClick={() => onSelect(chat)}
      className={`group flex items-center gap-3 px-3 py-2.5 rounded-lg cursor-pointer transition-colors duration-150 ${
        isSelected
          ? 'bg-[#2C2C2C]'
          : 'bg-transparent hover:bg-[#252525]'
      }`}
    >
      <div className="relative">
        {isGroup ? (
          <div className="w-11 h-11 rounded-full bg-[#2A2A2A] border border-[#383838] flex items-center justify-center">
            <Users className="w-5 h-5 text-[#C76029]" />
          </div>
        ) : (
          <>
            <Avatar title={chat.title} size={44} />
            {chat.peerUser.isOnline && (
              <span className="absolute bottom-0 right-0 w-3 h-3 bg-[#4CAF50] rounded-full border-2 border-[#222222]" />
            )}
          </>
        )}
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-1 mb-0.5">
          <h4 className="text-[14px] font-semibold text-white truncate">
            {chat.title}
            {isGroup && memberCount !== undefined && (
              <span className="ml-1.5 text-[11px] font-normal text-[#808080]">
                {memberCount}
              </span>
            )}
          </h4>
          <span className="text-[11px] text-[#D1D1D1] shrink-0">
            {formatTime(chat.updatedAt)}
          </span>
        </div>

        <div className="flex items-center justify-between gap-2">
          <p className="text-[12px] text-[#A0A0A0] truncate">
            {truncatedPreview}
          </p>

          {chat.unreadCount > 0 && (
            <span className="shrink-0 min-w-[20px] h-[18px] px-1.5 flex items-center justify-center bg-[#C76029] text-white text-[10px] font-bold rounded-full">
              {chat.unreadCount}
            </span>
          )}
        </div>
      </div>
    </div>
  );
};
