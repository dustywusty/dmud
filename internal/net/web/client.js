import { setupThemeSync } from "./theme.js";

const RECONNECT_DELAY_MS = 4000;
const MAX_LINES = 1200;
const CLEAR_COMMANDS = new Set(["/clear", ":clear"]);

const outputEl = document.querySelector('[data-role="output"]');
const statusBadge = document.querySelector('[data-role="status"]');
const hostEl = document.querySelector('[data-role="host"]');
const reconnectButton = document.querySelector('[data-action="reconnect"]');
const form = document.querySelector('[data-role="form"]');
const input = document.querySelector('[data-role="input"]');
const sendButton = document.querySelector('.command-bar__send');

const params = new URLSearchParams(window.location.search);
const websocketUrl = resolveWebSocketUrl(params);

hostEl.textContent = websocketUrl;

const history = [];
let historyIndex = -1;
let socket;
let reconnectTimer = null;
let isManuallyClosed = false;
let queuedCommands = [];

setupThemeSync();
updateStatus("connecting");
connect();

reconnectButton.addEventListener("click", () => {
  isManuallyClosed = false;
  clearTimeout(reconnectTimer);
  reconnectTimer = null;
  connect(true);
});

form.addEventListener("submit", (event) => {
  event.preventDefault();
  const value = input.value;
  if (!value.trim()) {
    return;
  }
  if (CLEAR_COMMANDS.has(value.trim().toLowerCase())) {
    clearOutput();
    appendSystemLine("Cleared output.");
    input.value = "";
    history.push(value);
    historyIndex = history.length;
    return;
  }
  sendCommand(value);
  input.value = "";
  history.push(value);
  historyIndex = history.length;
});

input.addEventListener("keydown", (event) => {
  if (event.key === "ArrowUp") {
    event.preventDefault();
    navigateHistory(-1);
  } else if (event.key === "ArrowDown") {
    event.preventDefault();
    navigateHistory(1);
  } else if (event.key === "Escape") {
    input.value = "";
  }
});

function resolveWebSocketUrl(searchParams) {
  const explicitWs = searchParams.get("ws") || searchParams.get("endpoint");
  if (explicitWs) {
    return explicitWs;
  }
  const host = searchParams.get("host");
  const path = searchParams.get("path") || "/ws";
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  if (host) {
    return `${protocol}://${host}${path.startsWith("/") ? path : `/${path}`}`;
  }
  return `${protocol}://${window.location.host}${path}`;
}

function connect(isReconnect = false) {
  updateStatus("connecting");
  disableInput(true);
  try {
    socket = new WebSocket(websocketUrl);
  } catch (error) {
    appendSystemLine("Failed to create WebSocket connection.");
    scheduleReconnect();
    return;
  }

  socket.addEventListener("open", () => {
    updateStatus("connected");
    disableInput(false);
    if (queuedCommands.length > 0) {
      const pending = [...queuedCommands];
      queuedCommands = [];
      for (const command of pending) {
        socket.send(command);
      }
    }
    if (isReconnect) {
      appendSystemLine("Reconnected to the server.");
    }
  });

  socket.addEventListener("close", (event) => {
    disableInput(true);
    updateStatus(event.wasClean ? "disconnected" : "error");
    if (!isManuallyClosed) {
      scheduleReconnect();
    }
  });

  socket.addEventListener("error", () => {
    updateStatus("error");
  });

  socket.addEventListener("message", (event) => {
    appendMessage(event.data);
  });
}

function sendCommand(command) {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    appendSystemLine("Not connected. Queuing command.");
    queuedCommands.push(command);
    scheduleReconnect();
    return;
  }
  try {
    socket.send(command);
  } catch (error) {
    appendSystemLine("Failed to send command. Retrying soon.");
    queuedCommands.push(command);
    if (socket.readyState !== WebSocket.CONNECTING) {
      scheduleReconnect();
    }
  }
}

function appendMessage(rawMessage) {
  if (typeof rawMessage !== "string") {
    return;
  }
  const text = rawMessage.replace(/\r\n?/g, "\n");
  const segments = text.split("\n");
  const shouldScroll = nearBottom();

  for (const segment of segments) {
    if (segment === "" && !outputEl.children.length) {
      continue;
    }
    const line = document.createElement("div");
    line.className = "line";
    if (segment.trim().length === 0) {
      line.classList.add("line--blank");
    } else {
      line.textContent = segment;
    }
    outputEl.appendChild(line);
  }

  while (outputEl.children.length > MAX_LINES) {
    outputEl.removeChild(outputEl.firstChild);
  }

  if (shouldScroll) {
    scrollToBottom();
  }
}

function appendSystemLine(message) {
  const shouldScroll = nearBottom();
  const line = document.createElement("div");
  line.className = "line line--system";
  line.textContent = message;
  outputEl.appendChild(line);
  if (outputEl.children.length > MAX_LINES) {
    outputEl.removeChild(outputEl.firstChild);
  }
  if (shouldScroll) {
    scrollToBottom();
  }
}

function clearOutput() {
  outputEl.innerHTML = "";
}

function nearBottom() {
  return outputEl.scrollHeight - outputEl.scrollTop - outputEl.clientHeight < 24;
}

function scrollToBottom() {
  outputEl.scrollTop = outputEl.scrollHeight;
}

function scheduleReconnect() {
  if (reconnectTimer) {
    return;
  }
  reconnectTimer = window.setTimeout(() => {
    reconnectTimer = null;
    if (!isManuallyClosed) {
      connect(true);
    }
  }, RECONNECT_DELAY_MS);
}

function disableInput(disabled) {
  input.disabled = disabled;
  if (sendButton) {
    sendButton.disabled = disabled;
    sendButton.setAttribute("aria-disabled", String(disabled));
  }
}

function updateStatus(state) {
  statusBadge.dataset.status = state;
  switch (state) {
    case "connected":
      statusBadge.textContent = "Connected";
      break;
    case "connecting":
      statusBadge.textContent = "Connecting…";
      break;
    case "error":
      statusBadge.textContent = "Connection issue";
      break;
    default:
      statusBadge.textContent = "Disconnected";
      break;
  }
}

function navigateHistory(direction) {
  if (!history.length) {
    return;
  }
  if (historyIndex === -1) {
    historyIndex = history.length;
  }
  historyIndex += direction;
  if (historyIndex < 0) {
    historyIndex = 0;
  } else if (historyIndex > history.length) {
    historyIndex = history.length;
  }
  if (historyIndex === history.length) {
    input.value = "";
    return;
  }
  input.value = history[historyIndex] ?? "";
  window.requestAnimationFrame(() => {
    input.setSelectionRange(input.value.length, input.value.length);
  });
}

window.addEventListener("beforeunload", () => {
  isManuallyClosed = true;
  clearTimeout(reconnectTimer);
  reconnectTimer = null;
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.close(1000, "Page unloading");
  }
});
