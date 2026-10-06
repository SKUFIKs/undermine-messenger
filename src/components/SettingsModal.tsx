import React, { useState } from 'react';
import { Settings, Copy, Check, Server, ShieldCheck, Key } from 'lucide-react';

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  currentUser: {
    username: string;
    userId: string;
    publicKeyHex: string;
  };
  serverUrl: string;
  onSaveServerUrl: (url: string) => void;
  isConnected: boolean;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({
  isOpen,
  onClose,
  currentUser,
  serverUrl,
  onSaveServerUrl,
  isConnected,
}) => {
  const [copiedKey, setCopiedKey] = useState(false);
  const [copiedId, setCopiedId] = useState(false);
  const [urlInput, setUrlInput] = useState(serverUrl);

  if (!isOpen) return null;

  const handleCopyKey = () => {
    navigator.clipboard.writeText(currentUser.publicKeyHex);
    setCopiedKey(true);
    setTimeout(() => setCopiedKey(false), 2000);
  };

  const handleCopyId = () => {
    navigator.clipboard.writeText(currentUser.userId);
    setCopiedId(true);
    setTimeout(() => setCopiedId(false), 2000);
  };

  const handleSaveUrl = (e: React.FormEvent) => {
    e.preventDefault();
    onSaveServerUrl(urlInput.trim());
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-black/75 flex items-center justify-center p-4 z-50 animate-fade-in">
      <div className="bg-[#222222] border border-[#2D2D2D] rounded-xl w-full max-w-lg p-6 shadow-2xl">
        <div className="flex items-center justify-between mb-5 pb-3 border-b border-[#2D2D2D]">
          <h3 className="text-[17px] font-bold text-white flex items-center gap-2">
            <Settings className="w-5 h-5 text-[#C76029]" />
            Настройки UnderMine
          </h3>
          <button
            onClick={onClose}
            className="text-[#808080] hover:text-white text-lg font-bold p-1 cursor-pointer"
          >
            ✕
          </button>
        </div>

        <div className="space-y-5">
          {/* Cryptographic Identity */}
          <div className="bg-[#1A1A1A] p-4 rounded-lg border border-[#2D2D2D]">
            <div className="flex items-center justify-between mb-2">
              <span className="text-[12px] font-semibold text-[#D1D1D1] flex items-center gap-1.5">
                <Key className="w-3.5 h-3.5 text-[#C76029]" />
                Ваш публичный ключ Curve25519
              </span>
              <button
                type="button"
                onClick={handleCopyKey}
                className="text-[11px] text-[#C76029] hover:underline flex items-center gap-1 cursor-pointer"
              >
                {copiedKey ? <Check className="w-3 h-3 text-[#4CAF50]" /> : <Copy className="w-3 h-3" />}
                {copiedKey ? 'Скопировано!' : 'Копировать'}
              </button>
            </div>
            <p className="font-mono text-[11px] text-[#A0A0A0] break-all bg-[#141414] p-2.5 rounded border border-[#262626] select-all">
              {currentUser.publicKeyHex}
            </p>

            <div className="flex items-center justify-between mt-3 mb-1">
              <span className="text-[12px] font-semibold text-[#D1D1D1]">User ID (16 байт hex)</span>
              <button
                type="button"
                onClick={handleCopyId}
                className="text-[11px] text-[#C76029] hover:underline flex items-center gap-1 cursor-pointer"
              >
                {copiedId ? <Check className="w-3 h-3 text-[#4CAF50]" /> : <Copy className="w-3 h-3" />}
                {copiedId ? 'Скопировано!' : 'Копировать'}
              </button>
            </div>
            <p className="font-mono text-[11px] text-[#808080] break-all bg-[#141414] p-2 rounded border border-[#262626] select-all">
              {currentUser.userId}
            </p>
          </div>

          {/* Server URL Config */}
          <form onSubmit={handleSaveUrl} className="space-y-3">
            <div>
              <label className="block text-[12px] font-semibold text-[#D1D1D1] mb-1.5 flex items-center justify-between">
                <span className="flex items-center gap-1.5">
                  <Server className="w-3.5 h-3.5 text-[#C76029]" />
                  Адрес сервера (Go Backend / Relay)
                </span>
                <span
                  className={`text-[11px] ${
                    isConnected ? 'text-[#4CAF50]' : 'text-[#E05252]'
                  }`}
                >
                  {isConnected ? '● Подключено' : '○ Нет связи'}
                </span>
              </label>
              <input
                type="text"
                value={urlInput}
                onChange={(e) => setUrlInput(e.target.value)}
                placeholder="http://localhost:8080 или текущий хост"
                className="w-full bg-[#1A1A1A] border border-[#2D2D2D] focus:border-[#C76029] rounded-lg px-3.5 py-2 text-[13px] text-white outline-none"
              />
              <p className="text-[11px] text-[#707070] mt-1">
                Оставьте пустым для использования встроенного ретранслятора, либо укажите внешний
                адрес Go-сервера (по умолчанию <code className="text-[#A0A0A0]">http://localhost:8080</code>).
              </p>
            </div>

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-[#2D2D2D]">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 rounded-lg text-[13px] text-[#A0A0A0] hover:text-white cursor-pointer"
              >
                Закрыть
              </button>
              <button
                type="submit"
                className="bg-[#C76029] hover:bg-[#d46a30] text-white text-[13px] font-medium px-4 py-2 rounded-lg cursor-pointer"
              >
                Сохранить
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};
