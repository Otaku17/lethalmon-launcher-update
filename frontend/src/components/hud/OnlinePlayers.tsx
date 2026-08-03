import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Badge } from './Badge';
import './OnlinePlayers.css';

const PRESENCE_WS_URL = 'wss://api.lethalmon-fangame.com/api/v1/ws/presence';
const RECONNECT_DELAY_MS = 4000;

const noDrag = { '--wails-draggable': 'no-drag' } as React.CSSProperties;

/**
 * "Online players" badge shown in the title bar. Connects to the
 * launcher's presence websocket (api.lethalmon-fangame.com): every
 * connected launcher counts as one online player, and the total is
 * broadcast in real time by the server. Reconnects automatically if the
 * connection drops.
 *
 * <OnlinePlayers />
 */
export default function OnlinePlayers() {
  const { t } = useTranslation();
  const [count, setCount] = useState(0);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    let socket: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let cancelled = false;

    function connect() {
      socket = new WebSocket(PRESENCE_WS_URL);

      socket.onopen = () => setConnected(true);

      socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as { onlinePlayers?: number };
          if (typeof data.onlinePlayers === 'number') setCount(data.onlinePlayers);
        } catch {
          // unexpected payload, ignore
        }
      };

      socket.onclose = () => {
        setConnected(false);
        if (!cancelled) reconnectTimer = setTimeout(connect, RECONNECT_DELAY_MS);
      };

      socket.onerror = () => {
        socket?.close();
      };
    }

    connect();

    return () => {
      cancelled = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, []);

  const label = connected ? t('common.onlinePlayers', { count }) : t('common.offline');

  return (
    <div className="hud-online-players" style={noDrag}>
      <Badge variant={connected ? 'ok' : 'warn'}>{label}</Badge>
    </div>
  );
}
