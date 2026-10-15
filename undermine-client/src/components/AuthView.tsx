import React, { useState } from 'react';
import { ShieldCheck, KeyRound, Sparkles } from 'lucide-react';

interface AuthViewProps {
  onLogin: (username: string, password: string) => Promise<void>;
  onRegister: (username: string, email: string, password: string) => Promise<void>;
  isLoading: boolean;
  error: string | null;
  clearError: () => void;
}

export const AuthView: React.FC<AuthViewProps> = ({
  onLogin,
  onRegister,
  isLoading,
  error,
  clearError,
}) => {
  const [isRegister, setIsRegister] = useState(false);
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !password.trim()) {
      return;
    }
    if (isRegister && !email.trim()) {
      return;
    }

    if (isRegister) {
      await onRegister(username.trim(), email.trim(), password);
    } else {
      await onLogin(username.trim(), password);
    }
  };

  const toggleMode = () => {
    setIsRegister(!isRegister);
    clearError();
  };

  return (
    <div className="min-h-screen w-full bg-[#171717] flex flex-col items-center justify-center p-4">
      {/* Central Login Card */}
      <div className="w-full max-w-[400px] bg-[#222222] border border-[#2D2D2D] rounded-[16px] p-8 shadow-2xl">
        {/* Header */}
        <div className="text-center mb-6">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-[#C76029]/15 text-[#C76029] mb-3">
            <ShieldCheck className="w-7 h-7" />
          </div>
          <h1 className="text-[24px] font-bold tracking-tight text-white">UnderMine</h1>
          <p className="text-[13px] text-[#D1D1D1] mt-1">
            {isRegister ? 'Регистрация аккаунта' : 'Авторизация'}
          </p>
          <div className="flex items-center justify-center gap-1.5 mt-2 text-[11px] text-[#A0A0A0]">
            <KeyRound className="w-3.5 h-3.5 text-[#C76029]" />
          </div>
        </div>

        {/* Error message */}
        {error && (
          <div className="mb-5 p-3 rounded-lg bg-[#E05252]/15 border border-[#E05252]/40 text-[#E05252] text-[13px] leading-snug">
            ⚠️ {error}
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-[12px] font-medium text-[#D1D1D1] mb-1.5">
              Имя пользователя
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => {
                setUsername(e.target.value);
                if (error) clearError();
              }}
              placeholder="Имя пользователя (username)"
              disabled={isLoading}
              className="w-full bg-[#1A1A1A] border border-[#2D2D2D] focus:border-[#C76029] rounded-lg px-3.5 py-2.5 text-[14px] text-white placeholder-[#707070] outline-none transition-colors"
              required
            />
          </div>

          {isRegister && (
            <div>
              <label className="block text-[12px] font-medium text-[#D1D1D1] mb-1.5">
                Электронная почта
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                  if (error) clearError();
                }}
                placeholder="Электронная почта (email)"
                disabled={isLoading}
                className="w-full bg-[#1A1A1A] border border-[#2D2D2D] focus:border-[#C76029] rounded-lg px-3.5 py-2.5 text-[14px] text-white placeholder-[#707070] outline-none transition-colors"
                required
              />
            </div>
          )}

          <div>
            <label className="block text-[12px] font-medium text-[#D1D1D1] mb-1.5">
              Пароль
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value);
                if (error) clearError();
              }}
              placeholder="Пароль"
              disabled={isLoading}
              className="w-full bg-[#1A1A1A] border border-[#2D2D2D] focus:border-[#C76029] rounded-lg px-3.5 py-2.5 text-[14px] text-white placeholder-[#707070] outline-none transition-colors"
              required
            />
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="w-full mt-2 bg-[#C76029] hover:bg-[#d46a30] active:bg-[#b05321] disabled:opacity-50 text-white font-medium py-2.5 px-4 rounded-lg text-[14px] transition-colors shadow-sm flex items-center justify-center gap-2 cursor-pointer"
          >
            {isLoading ? (
              <span className="inline-block w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
            ) : isRegister ? (
              'Зарегистрироваться'
            ) : (
              'Войти'
            )}
          </button>

          <button
            type="button"
            onClick={toggleMode}
            disabled={isLoading}
            className="w-full text-center text-[13px] text-[#A0A0A0] hover:text-white pt-2 cursor-pointer transition-colors"
          >
            {isRegister ? 'Уже есть аккаунт? Войти' : 'Создать аккаунт'}
          </button>
        </form>
      </div>
    </div>
  );
};
