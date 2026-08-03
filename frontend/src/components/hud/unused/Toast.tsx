import { useCallback, useRef, useState } from 'react';
import './Toast.css';

const InfoIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="square">
    <circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" />
  </svg>
);

export interface ToastData {
  id: number;
  title: string;
  text: string;
}

export interface PushToastOptions {
  title: string;
  text: string;
  duration?: number;
}

/**
 * const { toasts, pushToast } = useToasts();
 * pushToast({ title: 'TRANSMISSION REÇUE', text: 'Relais COM-3 rétabli.' });
 * <ToastStack toasts={toasts} />
 */
export function useToasts() {
  const [toasts, setToasts] = useState<ToastData[]>([]);
  const counter = useRef(0);

  const pushToast = useCallback(({ title, text, duration = 4000 }: PushToastOptions) => {
    const id = ++counter.current;
    setToasts((prev) => [...prev, { id, title, text }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, duration);
  }, []);

  return { toasts, pushToast };
}

export interface ToastProps {
  title: string;
  text: string;
}

export function Toast({ title, text }: ToastProps) {
  return (
    <div className="hud-toast">
      <InfoIcon />
      <div>
        <div className="hud-toast__title">{title}</div>
        <div className="hud-toast__text">{text}</div>
      </div>
    </div>
  );
}

export interface ToastStackProps {
  toasts: ToastData[];
}

export default function ToastStack({ toasts }: ToastStackProps) {
  return (
    <div className="hud-toast-stack">
      {toasts.map((t) => (
        <Toast key={t.id} title={t.title} text={t.text} />
      ))}
    </div>
  );
}
