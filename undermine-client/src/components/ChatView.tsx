import React, { useState, useRef, useEffect } from 'react';
import type { Chat, Message, User, Group } from '../types';
import type { ToastType } from './Toast';
import { Avatar } from './Avatar';
import { ChatItem } from './ChatItem';
import { MessageBubble } from './MessageBubble';
import { SettingsModal } from './SettingsModal';
import {
  Search,
  LogOut,
  Send,
  Lock,
  UserPlus,
  ShieldCheck,
  CheckCircle2,
  RefreshCw,
  Users,
  Settings,
  Columns,
  Sparkles,
  UserMinus,
} from 'lucide-react';

interface ChatViewProps {
  currentUser: {
    id: string;
    username: string;
    token: string;
    publicKeyHex: string;
  };
  chats: Chat[];
  activeChat: Chat | null;
  messages: Message[];
  onSelectChat: (chat: Chat) => void;
  onSendMessage: (text: string) => void;
  onSearchUser: (username: string) => Promise<{ found: boolean; message?: string }>;
  onLogout: () => void;
  isConnected: boolean;
  onRefreshDirectory?: () => void;
  availableUsers?: { user_id: string; username: string; public_key: string }[];
  onStartChatWithUser?: (user: { user_id: string; username: string; public_key: string }) => void;
  serverUrl?: string;
  onSaveServerUrl?: (url: string) => void;
  isDualView?: boolean;
  onToggleDualView?: () => void;
  onNotify?: (type: ToastType, message: string) => void;
  groups?: Group[];
  onCreateGroup?: (name: string, memberIds: string[]) => Promise<void>;
  onAddGroupMember?: (groupId: string, userId: string) => Promise<void>;
  onRemoveGroupMember?: (groupId: string, userId: string) => Promise<void>;
  onLeaveGroup?: (groupId: string) => Promise<void>;
}

export const ChatView: React.FC<ChatViewProps> = ({
  currentUser,
  chats,
  activeChat,
  messages,
  onSelectChat,
  onSendMessage,
  onSearchUser,
  onLogout,
  isConnected,
  availableUsers = [],
  onStartChatWithUser,
  serverUrl = '',
  onSaveServerUrl = () => {},
  isDualView = false,
  onToggleDualView,
  onNotify,
  groups = [],
  onCreateGroup,
  onAddGroupMember,
  onRemoveGroupMember,
  onLeaveGroup,
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [inputText, setInputText] = useState('');
  const [isSearching, setIsSearching] = useState(false);
  const [searchMessage, setSearchMessage] = useState<string | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [addUsername, setAddUsername] = useState('');

  // Group creation modal
  const [showCreateGroupModal, setShowCreateGroupModal] = useState(false);
  const [groupName, setGroupName] = useState('');
  const [selectedMembers, setSelectedMembers] = useState<string[]>([]);
  const [isCreatingGroup, setIsCreatingGroup] = useState(false);

  // Group management modal
  const [showGroupManageModal, setShowGroupManageModal] = useState(false);
  const [groupManageSearch, setGroupManageSearch] = useState('');
  const [isAddingMember, setIsAddingMember] = useState(false);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Auto-scroll to bottom of messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Focus input when active chat changes
  useEffect(() => {
    if (activeChat) {
      inputRef.current?.focus();
    }
  }, [activeChat]);

  const handleSend = (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    const text = inputText.trim();
    if (!text || !activeChat) return;

    onSendMessage(text);
    setInputText('');
  };

  const handleSearchSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const query = addUsername.trim();
    if (!query) return;

    setIsSearching(true);
    setSearchMessage(null);
    try {
      const res = await onSearchUser(query);
      if (res.found) {
        setShowAddModal(false);
        setAddUsername('');
        onNotify?.('success', `Диалог с ${query} открыт`);
      } else {
        setSearchMessage(res.message || 'Пользователь не найден');
      }
    } finally {
      setIsSearching(false);
    }
  };

  const filteredChats = chats.filter((c) =>
    c.title.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="flex h-full w-full bg-[#171717] overflow-hidden select-none">
      {/* LEFT PANEL (Sidebar) - 32% width matching Go Offset 0.32 */}
      <div className="w-[300px] md:w-[340px] lg:w-[360px] bg-[#222222] border-r border-[#2D2D2D] flex flex-col shrink-0">
        {/* App Topbar / Current User */}
        <div className="p-3 border-b border-[#2D2D2D] flex items-center justify-between gap-2">
          <div className="flex items-center gap-2.5 min-w-0">
            <Avatar title={currentUser.username} size={36} />
            <div className="min-w-0">
              <div className="flex items-center gap-1.5">
                <span className="font-semibold text-[14px] text-white truncate">
                  {currentUser.username}
                </span>
                <span
                  className={`w-2 h-2 rounded-full shrink-0 ${
                    isConnected ? 'bg-[#4CAF50]' : 'bg-[#E05252]'
                  }`}
                  title={isConnected ? 'Подключен к WebSocket' : 'Офлайн'}
                />
              </div>
              <div className="text-[11px] text-[#A0A0A0] flex items-center gap-1">
                <Lock className="w-3 h-3 text-[#C76029]" />
                <span className="font-mono text-[10px] text-[#808080] truncate max-w-[110px]">
                  {currentUser.publicKeyHex.slice(0, 8)}...
                </span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-1 shrink-0">
            <button
              onClick={() => setShowCreateGroupModal(true)}
              className="p-1.5 hover:bg-[#2C2C2C] text-[#D1D1D1] hover:text-white rounded-lg transition-colors cursor-pointer"
              title="Создать группу"
            >
              <Users className="w-4 h-4" />
            </button>
            <button
              onClick={() => setShowAddModal(true)}
              className="p-1.5 hover:bg-[#2C2C2C] text-[#D1D1D1] hover:text-white rounded-lg transition-colors cursor-pointer"
              title="Найти или добавить контакт"
            >
              <UserPlus className="w-4 h-4" />
            </button>
            <button
              onClick={() => setShowSettingsModal(true)}
              className="p-1.5 hover:bg-[#2C2C2C] text-[#D1D1D1] hover:text-white rounded-lg transition-colors cursor-pointer"
              title="Настройки ключей и сервера"
            >
              <Settings className="w-4 h-4" />
            </button>
            <button
              onClick={onLogout}
              className="p-1.5 hover:bg-[#2C2C2C] text-[#D1D1D1] hover:text-[#E05252] rounded-lg transition-colors cursor-pointer"
              title="Выход"
            >
              <LogOut className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Search Bar */}
        <div className="p-3 border-b border-[#2D2D2D]">
          <div className="relative flex items-center">
            <Search className="w-4 h-4 text-[#808080] absolute left-3 pointer-events-none" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Поиск диалогов..."
              className="w-full bg-[#171717] border border-[#2D2D2D] focus:border-[#C76029] rounded-lg pl-9 pr-3 py-1.5 text-[13px] text-white placeholder-[#707070] outline-none transition-colors"
            />
          </div>
        </div>

        {/* Chats List */}
        <div className="flex-1 overflow-y-auto p-2 space-y-1">
          {filteredChats.length > 0 ? (
            filteredChats.map((chat) => {
              const isGroup = groups.some((g) => g.id === chat.id);
              const group = groups.find((g) => g.id === chat.id);
              return (
                <ChatItem
                  key={chat.id}
                  chat={chat}
                  isSelected={activeChat?.id === chat.id}
                  onSelect={onSelectChat}
                  isGroup={isGroup}
                  memberCount={group?.members.length}
                />
              );
            })
          ) : (
            <div className="py-8 px-4 text-center">
              <p className="text-[13px] text-[#808080] mb-3">
                {searchQuery ? 'Диалогов не найдено' : 'Нет активных диалогов'}
              </p>
              <button
                onClick={() => setShowAddModal(true)}
                className="inline-flex items-center gap-1.5 text-[12px] text-[#C76029] hover:underline cursor-pointer"
              >
                <UserPlus className="w-3.5 h-3.5" />
                <span>Найти пользователя</span>
              </button>
            </div>
          )}
        </div>

        {/* Quick Contacts Bar (to quickly start chat with registered peers) */}
        {availableUsers.length > 0 && (
          <div className="p-2.5 border-t border-[#2D2D2D] bg-[#1E1E1E]">
            <div className="flex items-center justify-between text-[11px] text-[#808080] mb-1.5 px-1">
              <span className="flex items-center gap-1">
                <Users className="w-3 h-3 text-[#C76029]" />
                Доступные контакты:
              </span>
            </div>
            <div className="flex items-center gap-1.5 overflow-x-auto pb-1">
              {availableUsers
                .filter((u) => u.username.toLowerCase() !== currentUser.username.toLowerCase())
                .map((u) => (
                  <button
                    key={u.user_id}
                    onClick={() => onStartChatWithUser && onStartChatWithUser(u)}
                    className="shrink-0 flex items-center gap-1.5 bg-[#2A2A2A] hover:bg-[#333333] border border-[#383838] px-2.5 py-1 rounded-full text-[12px] text-white transition-colors cursor-pointer"
                  >
                    <Avatar title={u.username} size={18} />
                    <span className="capitalize">{u.username}</span>
                  </button>
                ))}
            </div>
          </div>
        )}
      </div>

      {/* RIGHT PANEL (Chat Content Area) */}
      <div className="flex-1 flex flex-col bg-[#171717] min-w-0 relative">
        {activeChat ? (
          <>
            {/* Chat Header */}
            <div className="h-[60px] bg-[#222222] border-b border-[#2D2D2D] px-5 flex items-center justify-between shrink-0 shadow-sm">
              <div className="flex items-center gap-3 min-w-0">
                <Avatar title={activeChat.title} size={40} />
                <div className="min-w-0">
                  <h3 className="text-[15px] font-bold text-white truncate">
                    {activeChat.title}
                  </h3>
                  <div className="flex items-center gap-2">
                    <span
                      className={`text-[11px] ${
                        activeChat.peerUser.isOnline
                          ? 'text-[#4CAF50] font-medium'
                          : 'text-[#D1D1D1]'
                      }`}
                    >
                      {activeChat.peerUser.isOnline ? '● в сети' : 'не в сети'}
                    </span>
                    <span className="text-[10px] text-[#666666]">•</span>
                    <span className="text-[10px] text-[#A0A0A0] flex items-center gap-1">
                      <ShieldCheck className="w-3 h-3 text-[#C76029]" />
                    </span>
                  </div>
                </div>
              </div>

              {/* Status info */}
              <div className="flex items-center gap-2">
                {activeChat && groups.some((g) => g.id === activeChat.id) && (
                  <button
                    onClick={() => {
                      setGroupManageSearch('');
                      setShowGroupManageModal(true);
                    }}
                    className="flex items-center gap-1.5 text-[11px] text-[#808080] hover:text-white bg-[#1A1A1A] hover:bg-[#2A2A2A] px-3 py-1.5 rounded-full border border-[#2D2D2D] transition-colors cursor-pointer"
                    title="Управление группой"
                  >
                    <Users className="w-3.5 h-3.5" />
                    <span>{groups.find((g) => g.id === activeChat.id)?.members.length || 0} уч.</span>
                  </button>
                )}
                <div className="hidden sm:flex items-center gap-2 text-[11px] text-[#808080] bg-[#1A1A1A] px-3 py-1.5 rounded-full border border-[#2D2D2D]">
                  <CheckCircle2 className="w-3.5 h-3.5 text-[#4CAF50]" />
                </div>
              </div>
            </div>

            {/* Messages Scroll Area */}
            <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-1">
              {messages.length === 0 ? (
                <div className="h-full flex flex-col items-center justify-center text-center text-[#808080] p-4">
                  <div className="w-12 h-12 rounded-full bg-[#222222] flex items-center justify-center mb-3 text-[#C76029]">
                    <Lock className="w-6 h-6" />
                  </div>
                  <p className="text-[14px] text-white font-medium mb-1">
                    
                  </p>
                  <p className="text-[12px] max-w-[320px] text-[#808080]">
                    
                  </p>
                </div>
              ) : (
                <>
                  <div className="text-center my-3">
                    <span className="inline-block bg-[#222222] border border-[#2D2D2D] px-3 py-1 rounded-full text-[11px] text-[#A0A0A0]">
                      
                    </span>
                  </div>
                  {messages.map((msg) => (
                    <MessageBubble key={msg.id} message={msg} />
                  ))}
                  <div ref={messagesEndRef} />
                </>
              )}
            </div>

            {/* Input Bar */}
            <div className="bg-[#222222] border-t border-[#2D2D2D] p-3 shrink-0">
              <form onSubmit={handleSend} className="flex items-center gap-2 max-w-5xl mx-auto">
                <input
                  ref={inputRef}
                  type="text"
                  value={inputText}
                  onChange={(e) => setInputText(e.target.value)}
                  placeholder="Написать сообщение... (Enter для отправки)"
                  className="flex-1 bg-[#171717] border border-[#2D2D2D] focus:border-[#C76029] rounded-lg px-4 py-2.5 text-[13px] text-white placeholder-[#707070] outline-none transition-colors"
                />
                <button
                  type="submit"
                  disabled={!inputText.trim()}
                  className="bg-[#C76029] hover:bg-[#d46a30] active:bg-[#b05321] disabled:opacity-40 disabled:hover:bg-[#C76029] text-white font-medium px-4 py-2.5 rounded-lg text-[13px] transition-colors flex items-center gap-1.5 shrink-0 cursor-pointer"
                >
                  <Send className="w-4 h-4" />
                  <span className="hidden sm:inline">Отправить</span>
                </button>
              </form>
            </div>
          </>
        ) : (
          /* Empty Placeholder matching Go client */
          <div className="h-full flex flex-col items-center justify-center p-6 text-center select-none">
            <div className="w-16 h-16 rounded-full bg-[#222222] border border-[#2D2D2D] flex items-center justify-center mb-4 text-[#C76029]">
              <ShieldCheck className="w-8 h-8" />
            </div>
            <h2 className="text-[18px] font-bold text-white mb-1">UnderMine Messenger</h2>
            <p className="text-[14px] text-[#D1D1D1] max-w-sm">
              Выберите чат, чтобы начать общение
            </p>
          </div>
        )}
      </div>

      {/* Add User Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center p-4 z-50 animate-fade-in">
          <div className="bg-[#222222] border border-[#2D2D2D] rounded-xl w-full max-w-md p-6 shadow-2xl">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-[16px] font-bold text-white flex items-center gap-2">
                <UserPlus className="w-5 h-5 text-[#C76029]" />
                Поиск собеседника
              </h3>
              <button
                onClick={() => {
                  setShowAddModal(false);
                  setSearchMessage(null);
                }}
                className="text-[#808080] hover:text-white text-lg font-bold p-1 cursor-pointer"
              >
                ✕
              </button>
            </div>

            <p className="text-[13px] text-[#A0A0A0] mb-4">
              Введите имя пользователя для поиска:
            </p>

            <form onSubmit={handleSearchSubmit} className="space-y-4">
              <div>
                <input
                  type="text"
                  value={addUsername}
                  onChange={(e) => setAddUsername(e.target.value)}
                  placeholder="Имя пользователя"
                  className="w-full bg-[#171717] border border-[#2D2D2D] focus:border-[#C76029] rounded-lg px-3.5 py-2.5 text-[14px] text-white placeholder-[#707070] outline-none"
                  autoFocus
                />
              </div>

              {searchMessage && (
                <div className="p-2.5 rounded-lg bg-[#E05252]/15 border border-[#E05252]/30 text-[#E05252] text-[12px]">
                  ⚠️ {searchMessage}
                </div>
              )}

              <div className="flex items-center justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowAddModal(false)}
                  className="px-4 py-2 rounded-lg text-[13px] text-[#A0A0A0] hover:text-white cursor-pointer"
                >
                  Отмена
                </button>
                <button
                  type="submit"
                  disabled={isSearching || !addUsername.trim()}
                  className="bg-[#C76029] hover:bg-[#d46a30] disabled:opacity-50 px-4 py-2 rounded-lg text-[13px] font-medium text-white transition-colors cursor-pointer"
                >
                  {isSearching ? 'Поиск...' : 'Найти и открыть чат'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Create Group Modal */}
      {showCreateGroupModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center p-4 z-50 animate-fade-in">
          <div className="bg-[#222222] border border-[#2D2D2D] rounded-xl w-full max-w-md p-6 shadow-2xl">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-[16px] font-bold text-white flex items-center gap-2">
                <Users className="w-5 h-5 text-[#C76029]" />
                Создать группу
              </h3>
              <button
                onClick={() => {
                  setShowCreateGroupModal(false);
                  setGroupName('');
                  setSelectedMembers([]);
                }}
                className="text-[#808080] hover:text-white text-lg font-bold p-1 cursor-pointer"
              >
                ✕
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-[12px] font-medium text-[#D1D1D1] mb-1.5">
                  Название группы
                </label>
                <input
                  type="text"
                  value={groupName}
                  onChange={(e) => setGroupName(e.target.value)}
                  placeholder="Например: Команда проекта"
                  className="w-full bg-[#171717] border border-[#2D2D2D] focus:border-[#C76029] rounded-lg px-3.5 py-2.5 text-[14px] text-white placeholder-[#707070] outline-none"
                  autoFocus
                />
              </div>

              <div>
                <label className="block text-[12px] font-medium text-[#D1D1D1] mb-1.5">
                  Участники
                </label>
                <div className="max-h-48 overflow-y-auto space-y-1">
                  {availableUsers
                    .filter((u) => u.user_id !== currentUser.id)
                    .map((u) => (
                      <label
                        key={u.user_id}
                        className="flex items-center gap-2.5 p-2 rounded-lg hover:bg-[#2A2A2A] cursor-pointer transition-colors"
                      >
                        <input
                          type="checkbox"
                          checked={selectedMembers.includes(u.user_id)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSelectedMembers((prev) => [...prev, u.user_id]);
                            } else {
                              setSelectedMembers((prev) => prev.filter((id) => id !== u.user_id));
                            }
                          }}
                          className="w-4 h-4 rounded accent-[#C76029]"
                        />
                        <Avatar title={u.username} size={24} />
                        <span className="text-[13px] text-white">{u.username}</span>
                      </label>
                    ))}
                </div>
              </div>

              <div className="flex items-center justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => {
                    setShowCreateGroupModal(false);
                    setGroupName('');
                    setSelectedMembers([]);
                  }}
                  className="px-4 py-2 rounded-lg text-[13px] text-[#A0A0A0] hover:text-white cursor-pointer"
                >
                  Отмена
                </button>
                <button
                  onClick={async () => {
                    if (!groupName.trim() || selectedMembers.length === 0) return;
                    setIsCreatingGroup(true);
                    try {
                      await onCreateGroup?.(groupName.trim(), selectedMembers);
                      setShowCreateGroupModal(false);
                      setGroupName('');
                      setSelectedMembers([]);
                    } finally {
                      setIsCreatingGroup(false);
                    }
                  }}
                  disabled={isCreatingGroup || !groupName.trim() || selectedMembers.length === 0}
                  className="bg-[#C76029] hover:bg-[#d46a30] disabled:opacity-50 px-4 py-2 rounded-lg text-[13px] font-medium text-white transition-colors cursor-pointer"
                >
                  {isCreatingGroup ? 'Создание...' : 'Создать'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Group Management Modal */}
      {showGroupManageModal && activeChat && groups.some((g) => g.id === activeChat.id) && (() => {
        const group = groups.find((g) => g.id === activeChat.id)!;
        const isOwner = group.ownerId === currentUser.id;

        return (
          <div className="fixed inset-0 bg-black/70 flex items-center justify-center p-4 z-50 animate-fade-in">
            <div className="bg-[#222222] border border-[#2D2D2D] rounded-xl w-full max-w-md p-6 shadow-2xl">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-[16px] font-bold text-white flex items-center gap-2">
                  <Users className="w-5 h-5 text-[#C76029]" />
                  {group.name}
                </h3>
                <button
                  onClick={() => setShowGroupManageModal(false)}
                  className="text-[#808080] hover:text-white text-lg font-bold p-1 cursor-pointer"
                >
                  ✕
                </button>
              </div>

              {/* Members list */}
              <div className="mb-4">
                <label className="block text-[12px] font-medium text-[#D1D1D1] mb-1.5">
                  Участники ({group.members.length})
                </label>
                <div className="max-h-40 overflow-y-auto space-y-1">
                  {group.members.map((m) => (
                    <div
                      key={m.userId}
                      className="flex items-center justify-between p-2 rounded-lg hover:bg-[#2A2A2A] transition-colors"
                    >
                      <div className="flex items-center gap-2.5">
                        <Avatar title={m.username} size={24} />
                        <span className="text-[13px] text-white">
                          {m.username}
                          {m.userId === group.ownerId && (
                            <span className="ml-1.5 text-[10px] text-[#C76029]">(владелец)</span>
                          )}
                          {m.userId === currentUser.id && (
                            <span className="ml-1.5 text-[10px] text-[#808080]">(вы)</span>
                          )}
                        </span>
                      </div>
                      {isOwner && m.userId !== currentUser.id && (
                        <button
                          onClick={async () => {
                            await onRemoveGroupMember?.(group.id, m.userId);
                          }}
                          className="p-1 text-[#808080] hover:text-[#E05252] transition-colors cursor-pointer"
                          title="Удалить из группы"
                        >
                          <UserMinus className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {/* Add member */}
              {isOwner && (
                <div className="mb-4">
                  <label className="block text-[12px] font-medium text-[#D1D1D1] mb-1.5">
                    Добавить участника
                  </label>
                  <div className="flex gap-2">
                    <select
                      value={groupManageSearch}
                      onChange={(e) => setGroupManageSearch(e.target.value)}
                      className="flex-1 bg-[#171717] border border-[#2D2D2D] focus:border-[#C76029] rounded-lg px-3 py-2 text-[13px] text-white outline-none"
                    >
                      <option value="">Выберите пользователя...</option>
                      {availableUsers
                        .filter(
                          (u) =>
                            u.user_id !== currentUser.id &&
                            !group.members.some((m) => m.userId === u.user_id)
                        )
                        .map((u) => (
                          <option key={u.user_id} value={u.user_id}>
                            {u.username}
                          </option>
                        ))}
                    </select>
                    <button
                      onClick={async () => {
                        if (!groupManageSearch) return;
                        setIsAddingMember(true);
                        try {
                          await onAddGroupMember?.(group.id, groupManageSearch);
                          setGroupManageSearch('');
                        } finally {
                          setIsAddingMember(false);
                        }
                      }}
                      disabled={isAddingMember || !groupManageSearch}
                      className="bg-[#C76029] hover:bg-[#d46a30] disabled:opacity-50 px-3 py-2 rounded-lg text-[13px] font-medium text-white transition-colors cursor-pointer shrink-0"
                    >
                      <UserPlus className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              )}

              {/* Leave group */}
              <div className="flex justify-end pt-2 border-t border-[#2D2D2D]">
                <button
                  onClick={async () => {
                    await onLeaveGroup?.(group.id);
                    setShowGroupManageModal(false);
                  }}
                  className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-[13px] text-[#E05252] hover:bg-[#E05252]/10 transition-colors cursor-pointer"
                >
                  <LogOut className="w-3.5 h-3.5" />
                  Покинуть группу
                </button>
              </div>
            </div>
          </div>
        );
      })()}

      {/* Settings Modal */}
      <SettingsModal
        isOpen={showSettingsModal}
        onClose={() => setShowSettingsModal(false)}
        currentUser={{
          username: currentUser.username,
          userId: currentUser.id,
          publicKeyHex: currentUser.publicKeyHex,
        }}
        serverUrl={serverUrl}
        onSaveServerUrl={onSaveServerUrl}
        isConnected={isConnected}
      />
    </div>
  );
};
