"use client";

import { useEffect, useRef, useCallback } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import { getToken } from "@/lib/api";

interface Props {
  serverId: string;
  serverName?: string;
}

export default function TerminalClient({ serverId }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef      = useRef<Terminal | null>(null);
  const wsRef        = useRef<WebSocket | null>(null);
  const fitRef       = useRef<FitAddon | null>(null);

  const connect = useCallback(() => {
    if (!containerRef.current) return;

    // Destroy previous instance if reconnecting.
    wsRef.current?.close();
    termRef.current?.dispose();

    const term = new Terminal({
      theme: {
        background:   "#0d1117",
        foreground:   "#e6edf3",
        cursor:       "#e6edf3",
        black:        "#484f58",
        red:          "#ff7b72",
        green:        "#3fb950",
        yellow:       "#d29922",
        blue:         "#58a6ff",
        magenta:      "#bc8cff",
        cyan:         "#39c5cf",
        white:        "#b1bac4",
        brightBlack:  "#6e7681",
        brightRed:    "#ffa198",
        brightGreen:  "#56d364",
        brightYellow: "#e3b341",
        brightBlue:   "#79c0ff",
        brightMagenta:"#d2a8ff",
        brightCyan:   "#56d4dd",
        brightWhite:  "#f0f6fc",
      },
      fontSize:       14,
      fontFamily:     "'Fira Code', 'Cascadia Code', 'JetBrains Mono', Menlo, Monaco, monospace",
      fontWeight:     "normal",
      cursorBlink:    true,
      cursorStyle:    "bar",
      scrollback:     5000,
      allowProposedApi: true,
    });

    const fitAddon      = new FitAddon();
    const webLinksAddon = new WebLinksAddon();
    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    term.open(containerRef.current);
    fitAddon.fit();
    term.focus();

    termRef.current = term;
    fitRef.current  = fitAddon;

    // Build WebSocket URL.
    const token = getToken();
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const url   = `${proto}//${window.location.host}/api/v1/servers/${serverId}/terminal` +
                  (token ? `?token=${encodeURIComponent(token)}` : "");

    const ws = new WebSocket(url);
    ws.binaryType = "arraybuffer";
    wsRef.current = ws;

    ws.onopen = () => {
      // Send initial terminal size.
      ws.send(JSON.stringify({ cols: term.cols, rows: term.rows }));
      term.write("\x1b[90mConnected. Type to begin.\x1b[0m\r\n");
    };

    ws.onmessage = (e) => {
      if (e.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(e.data));
      } else {
        term.write(e.data as string);
      }
    };

    ws.onclose = (e) => {
      term.write(`\r\n\x1b[33mConnection closed (${e.code}). Press any key to reconnect.\x1b[0m\r\n`);
    };

    ws.onerror = () => {
      term.write("\r\n\x1b[31mConnection error.\x1b[0m\r\n");
    };

    // Keyboard input → WebSocket.
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      } else if (ws.readyState === WebSocket.CLOSED) {
        // Any keypress after disconnect triggers a reconnect.
        connect();
      }
    });

    // Terminal resize → send to backend.
    term.onResize(({ cols, rows }) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ cols, rows }));
      }
    });
  }, [serverId]);

  // Mount & cleanup.
  useEffect(() => {
    connect();
    return () => {
      wsRef.current?.close();
      termRef.current?.dispose();
    };
  }, [connect]);

  // Fit terminal when the container resizes.
  useEffect(() => {
    if (!containerRef.current) return;
    const ro = new ResizeObserver(() => fitRef.current?.fit());
    ro.observe(containerRef.current);
    return () => ro.disconnect();
  }, []);

  return (
    <div
      ref={containerRef}
      className="h-full w-full"
      // xterm.js injects its own DOM into this div.
    />
  );
}
