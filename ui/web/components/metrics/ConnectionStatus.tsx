'use client';

interface ConnectionStatusProps {
  connected: boolean;
  error?: string | null;
  onReconnect?: () => void;
}

export function ConnectionStatus({ connected, error, onReconnect }: ConnectionStatusProps) {
  return (
    <div className="flex items-center gap-2">
      <div className="flex items-center gap-1.5">
        <div
          className={`h-2 w-2 rounded-full ${
            connected
              ? 'bg-emerald-500 animate-pulse'
              : error
              ? 'bg-red-500'
              : 'bg-amber-500 animate-pulse'
          }`}
        />
        <span className="text-xs text-zinc-500">
          {connected ? 'Live' : error ? 'Disconnected' : 'Connecting...'}
        </span>
      </div>
      {!connected && onReconnect && (
        <button
          onClick={onReconnect}
          className="text-xs text-blue-600 hover:text-blue-700 hover:underline"
        >
          Reconnect
        </button>
      )}
    </div>
  );
}

