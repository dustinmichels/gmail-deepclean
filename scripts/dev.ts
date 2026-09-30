#!/usr/bin/env bun
import { spawn, type Subprocess } from "bun";

// --- Configuration & Constants ---
const FRONTEND_URL = "http://localhost:5173";
const BACKEND_URL = "http://localhost:8080";
const MAX_LOG_BUFFER = 2000;

// ANSI styling
const ESC = "\x1b";
const RESET = `${ESC}[0m`;
const BOLD = `${ESC}[1m`;
const DIM = `${ESC}[2m`;
const GREEN = `${ESC}[32m`;
const BOLD_GREEN = `${ESC}[1;32m`;
const CYAN = `${ESC}[36m`;
const BOLD_CYAN = `${ESC}[1;36m`;
const YELLOW = `${ESC}[33m`;
const BOLD_YELLOW = `${ESC}[1;33m`;
const MAGENTA = `${ESC}[35m`;
const BOLD_MAGENTA = `${ESC}[1;35m`;
const RED = `${ESC}[31m`;
const BOLD_RED = `${ESC}[1;31m`;
const GRAY = `${ESC}[90m`;

// --- Terminal & String Utilities ---

// Strip ANSI escape codes
function stripAnsi(str: string): string {
  return str.replace(/\x1b\[[0-9;]*[a-zA-Z]/g, "");
}

// Compute visible terminal width of a string (accounting for ANSI & wide Unicode chars)
function strWidth(str: string): number {
  const clean = stripAnsi(str);
  let w = 0;
  for (const ch of clean) {
    const code = ch.codePointAt(0) || 0;
    // Common wide emoji / East Asian ranges
    if (
      (code >= 0x1f300 && code <= 0x1faff) ||
      (code >= 0x2600 && code <= 0x27bf) ||
      (code >= 0x2b50 && code <= 0x2b55) ||
      (code >= 0x231a && code <= 0x23f3) ||
      (code >= 0x1100 &&
        (code <= 0x115f ||
          code === 0x2329 ||
          code === 0x232a ||
          (code >= 0x2e80 && code <= 0xa4cf && code !== 0x303f) ||
          (code >= 0xac00 && code <= 0xd7a3) ||
          (code >= 0xf900 && code <= 0xfaff) ||
          (code >= 0xfe10 && code <= 0xfe19) ||
          (code >= 0xfe30 && code <= 0xfe6f) ||
          (code >= 0xff00 && code <= 0xff60) ||
          (code >= 0xffe0 && code <= 0xffe6)))
    ) {
      w += 2;
    } else if (code >= 0xfe00 && code <= 0xfe0f) {
      // Variation selector, width 0
    } else {
      w += 1;
    }
  }
  return w;
}

// Pad string on right to target visible width
function padRight(str: string, targetWidth: number): string {
  const current = strWidth(str);
  if (current >= targetWidth) return str;
  return str + " ".repeat(targetWidth - current);
}

// Slice ANSI string to maximum visible width
function sliceAnsi(str: string, maxWidth: number): string {
  if (maxWidth <= 0) return "";
  let visible = 0;
  let result = "";
  let inEscape = false;
  let escapeSeq = "";

  for (let i = 0; i < str.length; i++) {
    const ch = str[i];
    if (ch === "\x1b") {
      inEscape = true;
      escapeSeq = ch;
      continue;
    }
    if (inEscape) {
      escapeSeq += ch;
      if (/[a-zA-Z]/.test(ch)) {
        inEscape = false;
        result += escapeSeq;
        escapeSeq = "";
      }
      continue;
    }
    const code = ch.codePointAt(0) || 0;
    const charWidth =
      code >= 0x1f300 && code <= 0x1faff ? 2 : code >= 0xfe00 && code <= 0xfe0f ? 0 : 1;
    if (visible + charWidth > maxWidth) {
      break;
    }
    visible += charWidth;
    result += ch;
  }
  return result + RESET;
}

// Sanitize incoming log lines from child processes
function sanitizeLog(text: string): string {
  return text
    .replace(/\x1b\[[0-9;]*[Jj]/g, "") // remove clear-screen / clear-line
    .replace(/\x1b\[[0-9;]*[Hh]/g, (m) => (m.includes("H") || m === "\x1b[h" ? "" : m)) // remove cursor home
    .replace(/\x1b\[\?1049[hl]/g, "") // remove alternate screen codes
    .replace(/\x1b\[[0-9;]*r/g, ""); // remove scroll region resets
}

// --- CLI Arguments Parsing ---
type LayoutMode = "auto" | "top" | "split";
let userLayoutPreference: LayoutMode = "auto";

for (let i = 2; i < process.argv.length; i++) {
  const arg = process.argv[i];
  if (arg === "-h" || arg === "--help") {
    console.log(`
${BOLD}Gmail DeepClean Dev Server Runner${RESET}

${BOLD}Usage:${RESET}
  bun run scripts/dev.ts [options]
  mise run dev [-- options]

${BOLD}Options:${RESET}
  -l, --layout <auto|top|split>   Layout mode (default: auto)
                                  auto:  split if width >= 110, else top
                                  top:   fixed header banner with logs below
                                  split: two-column layout (banner left, logs right)
  -h, --help                      Show this help message

${BOLD}Shortcuts while running:${RESET}
  ${BOLD}[o]${RESET}      Open web app in browser (${FRONTEND_URL}/)
  ${BOLD}[t]${RESET}      Toggle layout (Top banner ⇄ Two-column split)
  ${BOLD}[r]${RESET}      Restart Go backend server (Air)
  ${BOLD}[c]${RESET}      Clear log history
  ${BOLD}[↑/↓]${RESET}    Scroll log history (PageUp/PageDown, Home/End)
  ${BOLD}[q]${RESET}      Quit dev servers cleanly
`);
    process.exit(0);
  }
  if (arg === "-l" || arg === "--layout") {
    const val = process.argv[++i];
    if (val === "top" || val === "split" || val === "auto") {
      userLayoutPreference = val;
    }
  } else if (arg.startsWith("--layout=")) {
    const val = arg.split("=")[1];
    if (val === "top" || val === "split" || val === "auto") {
      userLayoutPreference = val;
    }
  }
}

if (process.env.DEV_LAYOUT) {
  const envVal = process.env.DEV_LAYOUT.toLowerCase();
  if (envVal === "top" || envVal === "stacked") userLayoutPreference = "top";
  if (envVal === "split" || envVal === "two-col" || envVal === "twocol")
    userLayoutPreference = "split";
  if (envVal === "auto") userLayoutPreference = "auto";
}

// --- State ---
interface ServiceState {
  name: string;
  url: string;
  ready: boolean;
  statusText: string;
  error?: string;
}

const services: { frontend: ServiceState; backend: ServiceState } = {
  frontend: {
    name: "Frontend",
    url: FRONTEND_URL,
    ready: false,
    statusText: "Starting...",
  },
  backend: {
    name: "Backend",
    url: BACKEND_URL,
    ready: false,
    statusText: "Starting...",
  },
};

let currentLayout: "top" | "split" = "top";
let logLines: string[] = [];
let scrollOffset = 0; // 0 = at bottom (live auto-scroll)
let isShuttingDown = false;
let renderScheduled = false;

let serverProc: Subprocess | null = null;
let frontendProc: Subprocess | null = null;

// Determine active layout
function getEffectiveLayout(width: number): "top" | "split" {
  if (userLayoutPreference === "top") return "top";
  if (userLayoutPreference === "split") return "split";
  return width >= 110 ? "split" : "top";
}

// Get current timestamp formatted HH:MM:SS
function getTimestamp(): string {
  const now = new Date();
  return [
    now.getHours().toString().padStart(2, "0"),
    now.getMinutes().toString().padStart(2, "0"),
    now.getSeconds().toString().padStart(2, "0"),
  ].join(":");
}

// Add a log entry
function addLog(source: "server" | "vite" | "dev", text: string) {
  const clean = sanitizeLog(text).replace(/\r$/, "");
  if (!clean.trim()) return;

  const time = `${GRAY}${getTimestamp()}${RESET}`;
  let tag = "";
  if (source === "server") tag = `${MAGENTA}[server]${RESET}`;
  else if (source === "vite") tag = `${CYAN}[vite]  ${RESET}`;
  else tag = `${YELLOW}[dev]   ${RESET}`;

  const formattedLine = `${time} ${tag} ${clean}`;
  logLines.push(formattedLine);
  if (logLines.length > MAX_LOG_BUFFER) {
    logLines.shift();
  }

  // If scrolled up, stay scrolled; otherwise auto-scroll
  scheduleRender();
}

// --- Health Checking ---
async function checkHealth() {
  if (isShuttingDown) return;

  if (!services.frontend.ready) {
    try {
      const res = await fetch(FRONTEND_URL, {
        signal: AbortSignal.timeout(400),
      });
      if (res.status > 0) {
        services.frontend.ready = true;
        services.frontend.statusText = "Ready";
        scheduleRender();
      }
    } catch {}
  }

  if (!services.backend.ready) {
    try {
      const res = await fetch(BACKEND_URL, {
        signal: AbortSignal.timeout(400),
      });
      if (res.status > 0) {
        services.backend.ready = true;
        services.backend.statusText = "Ready";
        scheduleRender();
      }
    } catch {}
  }
}

// --- Process Management ---
function startServer() {
  if (serverProc) {
    try {
      serverProc.kill("SIGTERM");
    } catch {}
  }

  services.backend.ready = false;
  services.backend.statusText = "Starting...";
  services.backend.error = undefined;

  serverProc = spawn(["air"], {
    stdout: "pipe",
    stderr: "pipe",
    env: { ...process.env, PORT: "8080" },
  });

  streamOutput(serverProc.stdout, "server");
  streamOutput(serverProc.stderr, "server");

  serverProc.exited.then((code) => {
    if (!isShuttingDown) {
      services.backend.ready = false;
      services.backend.statusText = `Exited (${code})`;
      if (code !== 0) {
        services.backend.error = `Exit code ${code}`;
        addLog("dev", `${BOLD_RED}Go server exited with code ${code}${RESET}`);
      }
      scheduleRender();
    }
  });
}

function startFrontend() {
  services.frontend.ready = false;
  services.frontend.statusText = "Starting...";
  services.frontend.error = undefined;

  frontendProc = spawn(["bun", "run", "dev"], {
    cwd: "frontend",
    stdout: "pipe",
    stderr: "pipe",
    env: { ...process.env },
  });

  streamOutput(frontendProc.stdout, "vite");
  streamOutput(frontendProc.stderr, "vite");

  frontendProc.exited.then((code) => {
    if (!isShuttingDown) {
      services.frontend.ready = false;
      services.frontend.statusText = `Exited (${code})`;
      if (code !== 0) {
        services.frontend.error = `Exit code ${code}`;
        addLog("dev", `${BOLD_RED}Vite frontend exited with code ${code}${RESET}`);
      }
      scheduleRender();
    }
  });
}

async function streamOutput(
  stream: ReadableStream<Uint8Array> | number | null | undefined,
  source: "server" | "vite",
) {
  if (!stream || typeof stream === "number") return;
  const reader = stream.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() ?? "";
      for (const line of lines) {
        if (!process.stdout.isTTY) {
          // Plain stream fallback for non-TTY
          const time = getTimestamp();
          const tag = source === "server" ? "[server]" : "[vite]";
          console.log(`${time} ${tag} ${line}`);
        } else {
          addLog(source, line);
        }
      }
    }
  } catch {}
  if (buffer.length > 0) {
    if (!process.stdout.isTTY) {
      console.log(buffer);
    } else {
      addLog(source, buffer);
    }
  }
}

// --- Browser Launcher ---
function openBrowser() {
  addLog("dev", `Opening ${FRONTEND_URL}/ in default browser...`);
  const platform = process.platform;
  try {
    if (platform === "darwin") {
      spawn(["open", FRONTEND_URL]);
    } else if (platform === "win32") {
      spawn(["cmd", "/c", "start", FRONTEND_URL]);
    } else {
      spawn(["xdg-open", FRONTEND_URL]);
    }
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    addLog("dev", `${RED}Failed to open browser: ${msg}${RESET}`);
  }
}

// --- Render Engine ---

function scheduleRender() {
  if (!process.stdout.isTTY || isShuttingDown) return;
  if (renderScheduled) return;
  renderScheduled = true;
  queueMicrotask(() => {
    renderScheduled = false;
    render();
  });
}

function render() {
  if (!process.stdout.isTTY || isShuttingDown) return;

  const width = process.stdout.columns || 80;
  const height = process.stdout.rows || 24;
  currentLayout = getEffectiveLayout(width);

  if (currentLayout === "split") {
    renderSplit(width, height);
  } else {
    renderTop(width, height);
  }
}

// Status dot formatting
function formatStatusDot(ready: boolean, error?: string): string {
  if (error) return `${BOLD_RED}● Failed${RESET}`;
  if (ready) return `${BOLD_GREEN}● Ready${RESET}`;
  return `${BOLD_YELLOW}⏳ Starting...${RESET}`;
}

function makeBoxTop(title: string, width: number): string {
  const prefix = `┌─${title}`;
  const fillLen = Math.max(0, width - strWidth(prefix) - 1);
  return `${BOLD_GREEN}${prefix}${"─".repeat(fillLen)}┐${RESET}`;
}

function makeBoxBottom(width: number): string {
  return `${BOLD_GREEN}└${"─".repeat(Math.max(0, width - 2))}┘${RESET}`;
}

function makeBoxRow(content: string, width: number): string {
  const innerWidth = width - 4; // between "│ " and " │"
  return `${BOLD_GREEN}│${RESET} ${padRight(content, innerWidth)} ${BOLD_GREEN}│${RESET}`;
}

// --- Top Banner Layout ---
function renderTop(width: number, height: number) {
  const boxWidth = Math.min(width, 92);
  const innerWidth = boxWidth - 4; // between "│ " and " │"

  const feStatus = formatStatusDot(services.frontend.ready, services.frontend.error);
  const beStatus = formatStatusDot(services.backend.ready, services.backend.error);
  const allReady = services.frontend.ready && services.backend.ready;
  const overallBadge = allReady
    ? `${BOLD_GREEN}🚀 Dev Servers Ready${RESET}`
    : `${BOLD_YELLOW}⏳ Initializing Servers...${RESET}`;

  const line1Left = `  ${BOLD_CYAN}👉 ${FRONTEND_URL}/${RESET}  ${DIM}(Open in browser)${RESET}`;
  const line1Str = `${line1Left}${" ".repeat(Math.max(2, innerWidth - strWidth(line1Left) - strWidth(overallBadge)))}${overallBadge}`;
  const line2Str = `  Frontend: ${feStatus} ${DIM}(:5173)${RESET}   │   Backend: ${beStatus} ${DIM}(:8080)${RESET}   │   DB: ${BOLD_GREEN}● SQLite${RESET}`;
  const line3Str = `  ${DIM}[o] Open   [t] Split Layout   [r] Restart Go   [c] Clear   [↑/↓] Scroll   [q] Quit${RESET}`;

  const bannerLines = [
    makeBoxTop(" 🚀 Gmail DeepClean Dev ", boxWidth),
    makeBoxRow(line1Str, boxWidth),
    makeBoxRow(line2Str, boxWidth),
    makeBoxRow(line3Str, boxWidth),
    makeBoxBottom(boxWidth),
  ];

  // Divider bar with scroll state
  let scrollInfo = "";
  if (scrollOffset > 0) {
    scrollInfo = ` ${BOLD_YELLOW}[↑ Scrolled +${scrollOffset} lines — press ↓ or 'c' to return]${RESET} `;
  }
  const prefix = `─── Live Logs ${scrollInfo}`;
  const divBarLength = Math.max(0, width - strWidth(prefix));
  const dividerLine = `${GRAY}${prefix}${"─".repeat(divBarLength)}${RESET}`;
  // Log viewport
  const bannerTotalRows = bannerLines.length + 1; // +1 for divider
  const availableLogRows = Math.max(1, height - bannerTotalRows);

  const startIdx = Math.max(0, logLines.length - availableLogRows - scrollOffset);
  const endIdx = Math.max(0, logLines.length - scrollOffset);
  const visibleLogs = logLines.slice(startIdx, endIdx);

  // Pad logs up to availableLogRows if fewer
  const renderedLogRows: string[] = [];
  for (let i = 0; i < availableLogRows; i++) {
    const rawLog = visibleLogs[i];
    if (rawLog) {
      renderedLogRows.push(sliceAnsi(rawLog, width));
    } else {
      renderedLogRows.push("");
    }
  }

  // Assemble full screen buffer
  const out = [
    `${ESC}[H`, // cursor home
    ...bannerLines,
    dividerLine,
    ...renderedLogRows,
  ].join("\n");

  process.stdout.write(out + `${ESC}[J`); // clear any leftover lines below
}

// --- Two-Column Layout (Split Mode) ---
function renderSplit(width: number, height: number) {
  const leftWidth = 38;
  const innerLeftWidth = leftWidth - 4; // inside "│ " and " │"
  const rightWidth = Math.max(20, width - leftWidth - 3); // -3 for " │ "

  const feStatus = formatStatusDot(services.frontend.ready, services.frontend.error);
  const beStatus = formatStatusDot(services.backend.ready, services.backend.error);
  const allReady = services.frontend.ready && services.backend.ready;

  // Build left pane lines
  const leftPanel: string[] = [
    makeBoxTop(" 🚀 Gmail DeepClean ", leftWidth),
    makeBoxRow("", leftWidth),
    makeBoxRow(
      allReady
        ? `${BOLD_GREEN}🚀 Dev Servers Ready${RESET}`
        : `${BOLD_YELLOW}⏳ Starting Servers...${RESET}`,
      leftWidth,
    ),
    makeBoxRow("", leftWidth),
    makeBoxRow(`${BOLD_CYAN}👉 ${FRONTEND_URL}/${RESET}`, leftWidth),
    makeBoxRow(`   ${DIM}(Open in browser)${RESET}`, leftWidth),
    makeBoxRow("", leftWidth),
    makeBoxRow(`${BOLD}Services${RESET}`, leftWidth),
    makeBoxRow(`● Frontend  ${feStatus}`, leftWidth),
    makeBoxRow(`  ${DIM}${FRONTEND_URL}/${RESET}`, leftWidth),
    makeBoxRow(`● Backend   ${beStatus}`, leftWidth),
    makeBoxRow(`  ${DIM}${BACKEND_URL}/${RESET}`, leftWidth),
    makeBoxRow(`● Database  ${BOLD_GREEN}SQLite${RESET} ${DIM}(./data/app.db)${RESET}`, leftWidth),
    makeBoxRow("", leftWidth),
    makeBoxRow(`${BOLD}Shortcuts${RESET}`, leftWidth),
    makeBoxRow(`  ${BOLD}[o]${RESET} Open in browser`, leftWidth),
    makeBoxRow(`  ${BOLD}[t]${RESET} Stack layout (Top)`, leftWidth),
    makeBoxRow(`  ${BOLD}[r]${RESET} Restart Go backend`, leftWidth),
    makeBoxRow(`  ${BOLD}[c]${RESET} Clear logs`, leftWidth),
    makeBoxRow(`  ${BOLD}[↑/↓]${RESET} Scroll history`, leftWidth),
    makeBoxRow(`  ${BOLD}[q]${RESET} Quit dev servers`, leftWidth),
    makeBoxBottom(leftWidth),
  ];

  // Right column logs
  const availableLogRows = height;
  const startIdx = Math.max(0, logLines.length - availableLogRows - scrollOffset);
  const endIdx = Math.max(0, logLines.length - scrollOffset);
  const visibleLogs = logLines.slice(startIdx, endIdx);

  // Scroll notification header if scrolled
  let scrollBadge = "";
  if (scrollOffset > 0) {
    scrollBadge = `${BOLD_YELLOW}[↑ Scrolled +${scrollOffset} lines — press ↓ or 'c' to return]${RESET}`;
  }

  const outRows: string[] = [];
  for (let r = 0; r < height; r++) {
    // Left column string
    let leftCell = "";
    if (r < leftPanel.length) {
      leftCell = leftPanel[r];
    } else {
      leftCell = " ".repeat(leftWidth);
    }

    // Right column log string
    let rightCell = "";
    if (r === 0 && scrollBadge) {
      rightCell = scrollBadge;
    } else {
      const logIdx = scrollBadge ? r - 1 : r;
      const rawLog = visibleLogs[logIdx];
      if (rawLog) {
        rightCell = sliceAnsi(rawLog, rightWidth);
      }
    }

    outRows.push(`${leftCell} ${GRAY}│${RESET} ${rightCell}`);
  }

  // Assemble full screen buffer
  const out = `${ESC}[H` + outRows.join("\n");
  process.stdout.write(out + `${ESC}[J`);
}

// --- Keyboard Input Handling ---
function setupKeyboardInput() {
  if (process.stdin.isTTY) {
    try {
      process.stdin.setRawMode(true);
    } catch {}
  }
  process.stdin.resume();

  process.stdin.on("end", () => {
    shutdown();
  });

  process.stdin.on("data", (data: Buffer) => {
    const key = data.toString();

    // Ctrl+C or 'q' -> Exit
    if (key.includes("\u0003") || key === "q" || key === "Q" || key.trim() === "q") {
      shutdown();
      return;
    }
    // 't' or Tab -> Toggle layout mode
    if (key === "t" || key === "T" || key === "\t") {
      if (userLayoutPreference === "auto") {
        // Toggle away from current effective layout
        const currentEffective = getEffectiveLayout(process.stdout.columns || 80);
        userLayoutPreference = currentEffective === "top" ? "split" : "top";
      } else if (userLayoutPreference === "top") {
        userLayoutPreference = "split";
      } else {
        userLayoutPreference = "top";
      }
      addLog("dev", `Layout switched to ${BOLD}${userLayoutPreference.toUpperCase()}${RESET}`);
      scheduleRender();
      return;
    }

    // 'o' -> Open browser
    if (key === "o" || key === "O") {
      openBrowser();
      return;
    }

    // 'r' -> Restart Go server
    if (key === "r" || key === "R") {
      addLog("dev", `${BOLD_YELLOW}🔄 Restarting Go backend (Air)...${RESET}`);
      killProcessTree(serverProc, "SIGTERM");
      startServer();
      scheduleRender();
      return;
    }

    // 'c' -> Clear logs
    if (key === "c" || key === "C") {
      logLines = [];
      scrollOffset = 0;
      scheduleRender();
      return;
    }

    // Arrow Up / Scroll up
    if (key === `${ESC}[A`) {
      const height = process.stdout.rows || 24;
      const maxScroll = Math.max(0, logLines.length - (height - 6));
      scrollOffset = Math.min(scrollOffset + 1, maxScroll);
      scheduleRender();
      return;
    }

    // Arrow Down / Scroll down
    if (key === `${ESC}[B`) {
      scrollOffset = Math.max(0, scrollOffset - 1);
      scheduleRender();
      return;
    }

    // Page Up
    if (key === `${ESC}[5~`) {
      const height = process.stdout.rows || 24;
      const maxScroll = Math.max(0, logLines.length - (height - 6));
      scrollOffset = Math.min(scrollOffset + 10, maxScroll);
      scheduleRender();
      return;
    }

    // Page Down
    if (key === `${ESC}[6~`) {
      scrollOffset = Math.max(0, scrollOffset - 10);
      scheduleRender();
      return;
    }

    // Home
    if (key === `${ESC}[H` || key === `${ESC}[1~`) {
      const height = process.stdout.rows || 24;
      scrollOffset = Math.max(0, logLines.length - (height - 6));
      scheduleRender();
      return;
    }

    // End
    if (key === `${ESC}[F` || key === `${ESC}[4~`) {
      scrollOffset = 0;
      scheduleRender();
      return;
    }
  });
}

// --- Process Tree Cleanup ---
function killProcessTree(proc: Subprocess | null, signal: "SIGTERM" | "SIGKILL" = "SIGTERM") {
  if (!proc) return;
  const pid = proc.pid;
  try {
    spawn(["pkill", signal === "SIGKILL" ? "-9" : "-TERM", "-P", String(pid)]);
  } catch {}
  try {
    proc.kill(signal);
  } catch {}
}

// --- Graceful Shutdown ---
async function shutdown() {
  if (isShuttingDown) return;
  isShuttingDown = true;

  // Restore terminal state
  if (process.stdout.isTTY) {
    try {
      process.stdin.setRawMode(false);
      process.stdin.pause();
    } catch {}
    // Exit alternate buffer, show cursor
    process.stdout.write(`${ESC}[?1049l${ESC}[?25h`);
  }

  console.log(`\n${BOLD_YELLOW}Stopping dev servers...${RESET}`);

  const killPromises: Promise<number>[] = [];

  if (serverProc) {
    killProcessTree(serverProc, "SIGTERM");
    killPromises.push(serverProc.exited);
  }

  if (frontendProc) {
    killProcessTree(frontendProc, "SIGTERM");
    killPromises.push(frontendProc.exited);
  }

  // Wait briefly for graceful exit, fallback to SIGKILL if hanging
  const { promise: timeoutPromise, resolve: resolveTimeout } = Promise.withResolvers<void>();
  const timer = setTimeout(resolveTimeout, 1000);
  await Promise.race([Promise.all(killPromises), timeoutPromise]);
  clearTimeout(timer);

  killProcessTree(serverProc, "SIGKILL");
  killProcessTree(frontendProc, "SIGKILL");

  console.log(`${BOLD_GREEN}✨ Dev servers stopped cleanly.${RESET}\n`);
  process.exit(0);
}

// Register signals
process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);
process.on("exit", () => {
  if (process.stdout.isTTY) {
    process.stdout.write(`${ESC}[?1049l${ESC}[?25h`);
  }
});

// Resize handler
if (process.stdout.isTTY) {
  process.stdout.on("resize", () => {
    scheduleRender();
  });
}

// --- Main Entry ---
async function main() {
  if (process.stdout.isTTY) {
    // Enter alternate screen buffer and hide cursor
    process.stdout.write(`${ESC}[?1049h${ESC}[?25l`);
  }
  setupKeyboardInput();

  addLog("dev", `${BOLD}Starting Gmail DeepClean development environment...${RESET}`);

  // Launch processes
  startServer();
  startFrontend();

  // Poll health checks every 400ms until ready, then every 2000ms
  const healthInterval = setInterval(async () => {
    await checkHealth();
    if (services.frontend.ready && services.backend.ready) {
      clearInterval(healthInterval);
      // Keep checking occasionally in background
      setInterval(checkHealth, 3000);
    }
  }, 400);

  // Initial render
  scheduleRender();
}

main().catch((err) => {
  console.error("Fatal dev error:", err);
  shutdown();
});
