import React, { useEffect, useRef } from 'react';

const PLUGIN_ID = 'uk.org.optimis.test';

export function App() {
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const params = new URLSearchParams(window.location.search);
  const pupilId = params.get('pupil_id');

  useEffect(() => {
    const sendInit = () => {
      const contentWin = iframeRef.current?.contentWindow;
      if (!contentWin || !pupilId) {
        return;
      }
      contentWin.postMessage(
        {
          type: 'mis:init',
          theme: { primary: '#2563eb' },
          context: { pupil_id: pupilId },
          permitted_methods: ['get_pupil_record'],
        },
        '*'
      );
    };

    const handleMessage = async (event: MessageEvent) => {
      const msg = event.data;
      if (!msg || typeof msg !== 'object') {
        return;
      }

      if (msg.type === 'mis:ready') {
        sendInit();
        return;
      }

      if (msg.type === 'mis:rpc:request') {
        const { request_id, method, args } = msg;
        if (!request_id || !method) {
          throw new Error('Malformed mis:rpc:request: missing request_id or method');
        }

        const res = await fetch(`/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/${encodeURIComponent(method)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ args: args ?? {} }),
        });

        const payload = await res.json();
        iframeRef.current?.contentWindow?.postMessage(
          {
            type: 'mis:rpc:response',
            request_id,
            ok: Boolean(payload.ok),
            data: payload.data,
            error_code: payload.error_code,
          },
          '*'
        );
      }
    };

    window.addEventListener('message', handleMessage);
    return () => {
      window.removeEventListener('message', handleMessage);
    };
  }, [pupilId]);

  return (
    <div data-testid="host-shell" style={{ fontFamily: 'system-ui, sans-serif', padding: '24px' }}>
      <header style={{ marginBottom: '16px' }}>
        <h1 style={{ fontSize: '20px', margin: 0 }}>Optimis School MIS — Host Shell</h1>
      </header>
      <main>
        <iframe
          ref={iframeRef}
          data-testid={`plugin-iframe-${PLUGIN_ID}`}
          sandbox="allow-scripts"
          src={`/plugins/${PLUGIN_ID}/ui/index.html`}
          title="Test Plugin Micro-Frontend"
          style={{ width: '100%', height: '240px', border: '1px solid #d1d5db', borderRadius: '8px' }}
        />
      </main>
    </div>
  );
}
