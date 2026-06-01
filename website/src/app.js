// Cozy Canvas Notes - Main Logic

// DOM Elements
const viewport = document.getElementById('viewport');
const canvasBoard = document.getElementById('canvas-board');
const notesContainer = document.getElementById('notes-container');
const cursorEl = document.getElementById('custom-cursor');

// Help panel
const helpPanel = document.getElementById('help-panel');
const btnToggleHelp = document.getElementById('btn-toggle-help');
const btnCloseHelp = document.getElementById('btn-close-help');

// Settings panel
const settingsPanel = document.getElementById('settings-panel');
const btnToggleSettings = document.getElementById('btn-toggle-settings');
const btnCloseSettings = document.getElementById('btn-close-settings');
const settingPhysics = document.getElementById('setting-physics');
const settingSpring = document.getElementById('setting-spring');
const springValueDisplay = document.getElementById('spring-value');

// Authentication controls
const btnLogin = document.getElementById('btn-login');
const loginBtnText = document.getElementById('login-btn-text');
const loginModal = document.getElementById('login-modal');
const btnCloseLogin = document.getElementById('btn-close-login');
const loginForm = document.getElementById('login-form');

// Authentication inputs
const registerFields = document.getElementById('register-only-fields');
const loginUsernameInput = document.getElementById('login-username');
const loginWord1InputReal = document.getElementById('login-word1');
const loginWord2InputReal = document.getElementById('login-word2');
const loginEmailInput = document.getElementById('login-email');
const loginPasswordInput = document.getElementById('login-password');
const linkSwitchAuth = document.getElementById('link-switch-auth');
const modalTitle = document.getElementById('modal-title');
const modalSubtitle = document.getElementById('modal-subtitle');
const authSubmitBtn = document.getElementById('auth-submit-btn');

// Notification Toast
const cozyToast = document.getElementById('cozy-toast');

// Waybar mode toggle
const btnModeNotes = document.getElementById('mode-notes');
const btnModeEnv = document.getElementById('mode-env');

// Statsistics
const coordDisplay = document.getElementById('coord-display');
const zoomDisplay = document.getElementById('zoom-display');
const notesCountDisplay = document.getElementById('notes-count');

// Create dynamic SVG layer for note connections
const connectionsSvg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
connectionsSvg.setAttribute('id', 'connections-svg');
connectionsSvg.style.position = 'absolute';
connectionsSvg.style.top = '-250000px';
connectionsSvg.style.left = '-250000px';
connectionsSvg.style.width = '500000px';
connectionsSvg.style.height = '500000px';
connectionsSvg.style.pointerEvents = 'none';
canvasBoard.insertBefore(connectionsSvg, notesContainer);

// Aplication State
let zoom = 1.0;
let panX = 0;
let panY = 0;

let isPanning = false;
let startPanX = 0;
let startPanY = 0;
let mouseStartX = 0;
let mouseStartY = 0;

let notes = [];         // Personal notes list
let envNotes = [];      // Env variables list
let connections = [];    // Node connections list
let currentMode = 'notes'; // 'notes' or 'env'
let currentUser = null; // Username of active user
let authMode = 'login'; // 'login' or 'register'

let activeDragNoteId = null;
let noteDragStartX = 0;
let noteDragStartY = 0;
let noteMouseStartX = 0;
let noteMouseStartY = 0;

// Cursor interpolation state
let targetMouseX = window.innerWidth / 2;
let targetMouseY = window.innerHeight / 2;
let cursorX = targetMouseX;
let cursorY = targetMouseY;
const cursorLerpFactor = 0.15;

// Panning Modifier Key
let isSpacePressed = false;

// Toast Timer
let toastTimer = null;

// Settings & Connection State
let physicsEnabled = true;
let springStiffness = 0.08;

let isLinkingActive = false;
let linkSourceNoteId = null;
let linkPreviewLine = null;

let simulation = null;

// ==========================================================================
// Safe Storage Wrapper (Go API Integration with JWT Authentication)
// ==========================================================================

let authToken = null;

async function apiFetch(endpoint, method = 'GET', body = null) {
  const headers = {
    'Content-Type': 'application/json',
  };

  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }

  const options = {
    method,
    headers,
  };

  if (body) {
    options.body = JSON.stringify(body);
  }

  try {
    const response = await fetch(endpoint, options);

    if (!response.ok) {
      throw new Error(`API Error: ${response.status}`);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      return await response.json();
    }
    return null;
  } catch (error) {
    console.error(`[NETWORK ERROR] Request failed for ${endpoint}:`, error.message);
    throw error;
  }
}

const dbService = {
  async getNotes() {
    try {
      return await apiFetch('/api/notes');
    } catch (e) {
      console.warn('API unavailable. Falling back to offline mode. Data is not synchronized.');
      const data = localStorage.getItem('cozy-canvas-notes-data');
      return data ? JSON.parse(data) : [];
    }
  },

  async saveNotes(notesList) {
    try {
      await apiFetch('/api/notes', 'POST', notesList);
    } catch (e) {
      console.warn('API unavailable. Temporarily storing notes in local storage.');
      localStorage.setItem('cozy-canvas-notes-data', JSON.stringify(notesList));
    }
  },

  async getEnvNotes() {
    try {
      return await apiFetch('/api/env-notes');
    } catch (e) {
      console.warn('API unavailable. Loading default system configuration.');
      const data = localStorage.getItem('cozy-canvas-env-notes-data');
      if (data) return JSON.parse(data);

      const defaultEnv = [
        { id: 'env-USER', text: '⚙️ USER\n\nguest_devops', x: window.innerWidth / 2 - 85, y: window.innerHeight / 2 - 280, isEnv: true },
        { id: 'env-OS', text: '⚙️ OS\n\nLinux (NixOS)', x: window.innerWidth / 2 - 85 + 200, y: window.innerHeight / 2 - 85, isEnv: true },
        { id: 'env-WM', text: '⚙️ COMPOSITOR\n\nHyprland', x: window.innerWidth / 2 - 85 - 200, y: window.innerHeight / 2 - 85, isEnv: true }
      ];
      localStorage.setItem('cozy-canvas-env-notes-data', JSON.stringify(defaultEnv));
      return defaultEnv;
    }
  },

  async saveEnvNotes(envNotesList) {
    try {
      await apiFetch('/api/env-notes', 'POST', envNotesList);
    } catch (e) {
      console.warn('API unavailable. Saving environment notes locally.');
      localStorage.setItem('cozy-canvas-env-notes-data', JSON.stringify(envNotesList));
    }
  },

  async getConnections() {
    try {
      return await apiFetch('/api/connections');
    } catch (e) {
      const data = localStorage.getItem('cozy-canvas-connections-data');
      return data ? JSON.parse(data) : [];
    }
  },

  async saveConnections(connectionsList) {
    try {
      localStorage.setItem('cozy-canvas-connections-data', JSON.stringify(connectionsList));
      await apiFetch('/api/connections', 'POST', connectionsList);
    } catch (e) {
      console.warn('API unavailable. Saved connections locally.');
    }
  }
};

// ==========================================================================
// Application Initialization & Data Loading
// ==========================================================================

async function init() {
  // Session tokens are restored only from sessionStorage
  // to reduce persistence and limit exposure.
  currentUser = sessionStorage.getItem('cozy-canvas-user') || null;
  updateLoginUI();

  try { notes = await dbService.getNotes(); } catch (e) { notes = []; }
  try { envNotes = await dbService.getEnvNotes(); } catch (e) { envNotes = []; }
  try { connections = await dbService.getConnections(); } catch (e) { connections = []; }

  const isHelpVisible = localStorage.getItem('cozy-canvas-help-visible') !== 'false';
  if (!isHelpVisible) {
    helpPanel.classList.add('hidden');
  }

  physicsEnabled = localStorage.getItem('cozy-canvas-setting-physics') !== 'false';
  settingPhysics.checked = physicsEnabled;

  const savedSpring = localStorage.getItem('cozy-canvas-setting-spring');
  const springSliderVal = savedSpring !== null ? parseInt(savedSpring, 10) : 8;
  settingSpring.value = springSliderVal;
  springStiffness = springSliderVal / 100;
  springValueDisplay.textContent = springStiffness.toFixed(2);

  centerView();
  renderNotes();
  updateTransform();
  updateStatus();

  if (physicsEnabled) {
    startPhysicsSimulation();
  }

  animateCursor();
  setupEventListeners();
}

// ==========================================================================
// Toast Notification System
// ==========================================================================

function showToast(message) {
  if (toastTimer) {
    clearTimeout(toastTimer);
  }

  cozyToast.textContent = message;
  cozyToast.classList.remove('hidden');

  toastTimer = setTimeout(() => {
    cozyToast.classList.add('hidden');
  }, 3800);
}

// ==========================================================================
// Authentication UI State
// ==========================================================================

function updateLoginUI() {
  if (currentUser) {
    loginBtnText.textContent = currentUser;
    btnLogin.title = `Logged in as ${currentUser}. Click to sign out.`;
    btnLogin.classList.add('accent');
  } else {
    loginBtnText.textContent = 'Sign in';
    btnLogin.title = 'Sign to Cozy Cluster';
    btnLogin.classList.remove('accent');
  }

  if (currentMode === 'env') {
    renderNotes();
  }
}

// ==========================================================================
// Canvas Panning & Zooming
// ==========================================================================

function updateTransform() {
  canvasBoard.style.transform = `translate(${panX}px, ${panY}px) scale(${zoom})`;

  const vCenterX = (window.innerWidth / 2 - panX) / zoom;
  const vCenterY = (window.innerHeight / 2 - panY) / zoom;

  coordDisplay.textContent = `${Math.round(vCenterX)}, ${Math.round(vCenterY)}`;
  zoomDisplay.textContent = `${Math.round(zoom * 100)}%`;
}

function centerView() {
  zoom = 1.0;
  panX = 0;
  panY = 0;

  updateTransform();
}

function adjustZoom(factor, clientX, clientY) {
  const rect = viewport.getBoundingClientRect();
  const mouseX = clientX - rect.left;
  const mouseY = clientY - rect.top;

  const oldZoom = zoom;
  let newZoom = zoom * factor;

  newZoom = Math.max(0.15, Math.min(3.0, newZoom));

  panX = mouseX - (mouseX - panX) * (newZoom / oldZoom);
  panY = mouseY - (mouseY - panY) * (newZoom / oldZoom);
  zoom = newZoom;

  updateTransform();
}

function screenToCanvas(clientX, clientY) {
  const rect = viewport.getBoundingClientRect();
  const x = (clientX - rect.left - panX) / zoom;
  const y = (clientY - rect.top - panY) / zoom;
  return { x, y };
}

// ==========================================================================
// Note & Connection Rendering
// ==========================================================================

function renderNotes() {
  notesContainer.innerHTML = '';

  const activeNotes = currentMode === 'notes' ? notes : envNotes;

  activeNotes.forEach(note => {
    const noteEl = document.createElement('div');
    noteEl.className = 'cozy-note';
    if (note.isEnv) {
      noteEl.className += ' env-node';
    }
    noteEl.id = `note-${note.id}`;
    noteEl.style.left = `${note.x}px`;
    noteEl.style.top = `${note.y}px`;

    // Note Content Editor
    const textEl = document.createElement('div');
    textEl.className = 'note-text';
    textEl.contentEditable = 'true';
    textEl.textContent = note.text;
    textEl.addEventListener('input', async (e) => {
      note.text = e.target.textContent;
      if (currentMode === 'notes') {
        await dbService.saveNotes(notes);
      } else {
        await dbService.saveEnvNotes(envNotes);
      }
    });
    textEl.addEventListener('blur', async () => {
      if (!note.text.trim()) {
        note.text = 'Note...';
        textEl.textContent = 'Note...';
        if (currentMode === 'notes') {
          await dbService.saveNotes(notes);
        } else {
          await dbService.saveEnvNotes(envNotes);
        }
      }
    });

    // Delete Button
    const deleteBtn = document.createElement('button');
    deleteBtn.className = 'btn-delete-note';
    deleteBtn.innerHTML = '✕';
    deleteBtn.title = 'Delete Note';
    deleteBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      deleteNote(note.id);
    });
    noteEl.appendChild(deleteBtn);

    // Mouse Interaction
    noteEl.addEventListener('mousedown', (e) => {
      if (e.target.contentEditable === 'true') return;

      const isRightClick = e.button === 2;
      const isLinkingTriggered = isRightClick;

      if (isLinkingTriggered) {
        e.stopPropagation();
        e.preventDefault();

        startLinking(note.id, e.clientX, e.clientY);
        return;
      }

      if (e.button === 0 || e.button === 1) { // Left or middle click
        e.stopPropagation();
        e.preventDefault();

        activeDragNoteId = note.id;
        noteMouseStartX = e.clientX;
        noteMouseStartY = e.clientY;
        noteDragStartX = note.x;
        noteDragStartY = note.y;

        noteEl.classList.add('dragging');
        viewport.classList.add('dragging-active');

        if (physicsEnabled && simulation) {
          const activeNotesList = currentMode === 'notes' ? notes : envNotes;
          const d3Node = activeNotesList.find(n => n.id === note.id);
          if (d3Node) {
            d3Node.fx = d3Node.x;
            d3Node.fy = d3Node.y;
          }
        }
      }
    });

    noteEl.appendChild(textEl);
    notesContainer.appendChild(noteEl);
  });

  notesCountDisplay.textContent = activeNotes.length.toString();
  updateConnections();
}

function updateConnections() {
  connectionsSvg.innerHTML = '';

  const rOffset = 85;

  if (currentMode === 'notes') {
    connections.forEach(conn => {
      const sId = typeof conn.source === 'object' ? conn.source.id : conn.source;
      const tId = typeof conn.target === 'object' ? conn.target.id : conn.target;

      const n1 = notes.find(n => n.id === sId);
      const n2 = notes.find(n => n.id === tId);

      if (n1 && n2) {
        const x1 = n1.x + rOffset;
        const y1 = n1.y + rOffset;
        const x2 = n2.x + rOffset;
        const y2 = n2.y + rOffset;

        const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
        line.setAttribute('x1', x1 + 250000);
        line.setAttribute('y1', y1 + 250000);
        line.setAttribute('x2', x2 + 250000);
        line.setAttribute('y2', y2 + 250000);
        line.classList.add('connection-line');

        line.addEventListener('click', (e) => {
          e.stopPropagation();
          deleteConnection(sId, tId);
        });

        connectionsSvg.appendChild(line);
      }
    });
  } else {
    const maxDistance = 380;
    for (let i = 0; i < envNotes.length; i++) {
      for (let j = i + 1; j < envNotes.length; j++) {
        const n1 = envNotes[i];
        const n2 = envNotes[j];

        const x1 = n1.x + rOffset;
        const y1 = n1.y + rOffset;
        const x2 = n2.x + rOffset;
        const y2 = n2.y + rOffset;

        const dx = x1 - x2;
        const dy = y1 - y2;
        const dist = Math.sqrt(dx * dx + dy * dy);

        if (dist < maxDistance) {
          const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
          line.setAttribute('x1', x1 + 250000);
          line.setAttribute('y1', y1 + 250000);
          line.setAttribute('x2', x2 + 250000);
          line.setAttribute('y2', y2 + 250000);
          line.classList.add('connection-line');

          const opacity = (1 - dist / maxDistance) * 0.45;
          line.style.opacity = opacity;

          connectionsSvg.appendChild(line);
        }
      }
    }
  }
}

async function deleteConnection(sId, tId) {
  if (currentMode !== 'notes') return;
  connections = connections.filter(conn => {
    const currSId = typeof conn.source === 'object' ? conn.source.id : conn.source;
    const currTId = typeof conn.target === 'object' ? conn.target.id : conn.target;
    return !((currSId === sId && currTId === tId) || (currSId === tId && currTId === sId));
  });
  await dbService.saveConnections(connections);
  renderNotes();
  if (physicsEnabled && simulation) {
    updatePhysicsSimulationLinks();
    simulation.alpha(0.3).restart();
  }
  showToast('🌸 Connection removed.');
}

async function createConnection(sId, tId) {
  if (sId === tId) return;

  const exists = connections.some(conn => {
    const currSId = typeof conn.source === 'object' ? conn.source.id : conn.source;
    const currTId = typeof conn.target === 'object' ? conn.target.id : conn.target;
    return (currSId === sId && currTId === tId) || (currSId === tId && currTId === sId);
  });

  if (exists) {
    await deleteConnection(sId, tId);
    return;
  }

  connections.push({ source: sId, target: tId });
  await dbService.saveConnections(connections);

  if (physicsEnabled && simulation) {
    updatePhysicsSimulationLinks();
    simulation.alpha(0.5).restart();
  }

  showToast('🌸 Connection created!');
}

// ==========================================================================
// Note Actions
// ==========================================================================

async function addNote(x, y) {
  const id = Date.now().toString(36) + Math.random().toString(36).substr(2, 5);
  const isEnvMode = currentMode === 'env';

  const newNote = {
    id: id,
    text: isEnvMode ? '⚙️ NEW_VAR\n\nvalue...' : 'New Note 🌸\n\nClick here to edit...',
    x: x - 85,
    y: y - 85
  };

  if (isEnvMode) {
    newNote.isEnv = true;
    envNotes.push(newNote);
    await dbService.saveEnvNotes(envNotes);
  } else {
    notes.push(newNote);
    await dbService.saveNotes(notes);
  }
  renderNotes();

  if (physicsEnabled) {
    startPhysicsSimulation();
  }

  setTimeout(() => {
    const noteEl = document.getElementById(`note-${id}`);
    if (noteEl) {
      const textInput = noteEl.querySelector('.note-text');
      if (textInput) {
        textInput.focus();
        const range = document.createRange();
        range.selectNodeContents(textInput);
        const sel = window.getSelection();
        sel.removeAllRanges();
        sel.addRange(range);
      }
    }
  }, 50);
}

async function deleteNote(id) {
  const isEnvMode = currentMode === 'env';
  if (isEnvMode) {
    envNotes = envNotes.filter(n => n.id !== id);
    await dbService.saveEnvNotes(envNotes);
  } else {
    notes = notes.filter(n => n.id !== id);
    await dbService.saveNotes(notes);
  }

  connections = connections.filter(conn => {
    const sId = typeof conn.source === 'object' ? conn.source.id : conn.source;
    const tId = typeof conn.target === 'object' ? conn.target.id : conn.target;
    return sId !== id && tId !== id;
  });
  await dbService.saveConnections(connections);
  renderNotes();

  if (physicsEnabled) {
    startPhysicsSimulation();
  }
}

// ==========================================================================
// Event Binding
// ==========================================================================

function setupEventListeners() {
  window.addEventListener('keydown', (e) => {
    if (e.code === 'Space') {
      isSpacePressed = true;
      viewport.style.cursor = 'grab';
    }
    if (e.key === 'Escape') {
      if (!loginModal.classList.contains('hidden')) {
        loginModal.classList.add('hidden');
      } else if (!settingsPanel.classList.contains('hidden')) {
        settingsPanel.classList.add('hidden');
      } else if (!helpPanel.classList.contains('hidden')) {
        helpPanel.classList.add('hidden');
      } else {
        centerView();
      }
    }
  });

  window.addEventListener('keyup', (e) => {
    if (e.code === 'Space') {
      isSpacePressed = false;
      if (!isPanning) viewport.style.cursor = '';
    }
  });

  viewport.addEventListener('mousedown', (e) => {
    const isBackground = e.target.classList.contains('viewport') ||
      e.target.classList.contains('canvas-board') ||
      e.target.classList.contains('grid-background') ||
      e.target.id === 'connections-svg';

    if ((isBackground || isSpacePressed) && (e.button === 0 || e.button === 1)) {
      isPanning = true;
      mouseStartX = e.clientX;
      mouseStartY = e.clientY;
      startPanX = panX;
      startPanY = panY;
      viewport.style.cursor = 'grabbing';
      e.preventDefault();
    }
  });

  window.addEventListener('mousemove', (e) => {
    targetMouseX = e.clientX;
    targetMouseY = e.clientY;

    const target = e.target;
    const isClickable = target.closest('button, .cozy-note, a, [contenteditable="true"], line');
    if (isClickable) {
      viewport.classList.add('hovering-clickable');
    } else {
      viewport.classList.remove('hovering-clickable');
    }

    if (isPanning) {
      const dx = e.clientX - mouseStartX;
      const dy = e.clientY - mouseStartY;
      panX = startPanX + dx;
      panY = startPanY + dy;
      updateTransform();
    }

    else if (activeDragNoteId !== null) {
      const activeNotesList = currentMode === 'notes' ? notes : envNotes;
      const currentNote = activeNotesList.find(n => n.id === activeDragNoteId);
      if (currentNote) {
        const deltaScreenX = e.clientX - noteMouseStartX;
        const deltaScreenY = e.clientY - noteMouseStartY;

        const targetX = noteDragStartX + (deltaScreenX / zoom);
        const targetY = noteDragStartY + (deltaScreenY / zoom);

        currentNote.x = targetX;
        currentNote.y = targetY;

        if (physicsEnabled && simulation) {
          currentNote.fx = targetX;
          currentNote.fy = targetY;
          simulation.alphaTarget(0.1).restart();
        } else {
          const noteEl = document.getElementById(`note-${activeDragNoteId}`);
          if (noteEl) {
            noteEl.style.left = `${currentNote.x}px`;
            noteEl.style.top = `${currentNote.y}px`;
          }
          updateConnections();
        }
      }
    }

    else if (isLinkingActive) {
      updateLinkPreview(e.clientX, e.clientY);

      const hoveredNoteEl = e.target.closest('.cozy-note');
      document.querySelectorAll('.cozy-note').forEach(el => {
        el.classList.remove('link-target');
      });
      if (hoveredNoteEl) {
        const targetId = hoveredNoteEl.id.replace('note-', '');
        if (targetId !== linkSourceNoteId) {
          hoveredNoteEl.classList.add('link-target');
        }
      }
    }
  });

  window.addEventListener('mouseup', (e) => {
    if (isPanning) {
      isPanning = false;
      viewport.style.cursor = isSpacePressed ? 'grab' : '';
    }

    if (isLinkingActive) {
      finishLinking(e.clientX, e.clientY);
    }

    if (activeDragNoteId !== null) {
      if (physicsEnabled && simulation) {
        const activeNotesList = currentMode === 'notes' ? notes : envNotes;
        const currentNote = activeNotesList.find(n => n.id === activeDragNoteId);
        if (currentNote) {
          currentNote.fx = null;
          currentNote.fy = null;
        }
        simulation.alphaTarget(0);
      }

      const noteEl = document.getElementById(`note-${activeDragNoteId}`);
      if (noteEl) {
        noteEl.classList.remove('dragging');
      }
      activeDragNoteId = null;
      viewport.classList.remove('dragging-active');
      dbService.saveNotes(notes);
    }
  });

  viewport.addEventListener('wheel', (e) => {
    e.preventDefault();
    const zoomFactor = 1.08;
    const factor = e.deltaY < 0 ? zoomFactor : 1 / zoomFactor;
    adjustZoom(factor, e.clientX, e.clientY);
  }, { passive: false });

  viewport.addEventListener('dblclick', (e) => {
    if (currentMode !== 'notes') return;

    const isBackground = e.target.classList.contains('viewport') ||
      e.target.classList.contains('canvas-board') ||
      e.target.classList.contains('grid-background') ||
      e.target.id === 'connections-svg';

    if (isBackground) {
      const canvasCoords = screenToCanvas(e.clientX, e.clientY);
      addNote(canvasCoords.x, canvasCoords.y);
    }
  });

  btnModeNotes.addEventListener('click', () => {
    if (currentMode === 'notes') return;

    currentMode = 'notes';
    btnModeNotes.classList.add('active');
    btnModeEnv.classList.remove('active');

    renderNotes();
    updateStatus();

    if (physicsEnabled) startPhysicsSimulation();
    else stopPhysicsSimulation();
  });

  btnModeEnv.addEventListener('click', () => {
    if (currentMode === 'env') return;

    currentMode = 'env';
    btnModeEnv.classList.add('active');
    btnModeNotes.classList.remove('active');

    renderNotes();
    updateStatus();

    if (physicsEnabled) startPhysicsSimulation();
    else stopPhysicsSimulation();
  });

  btnToggleHelp.addEventListener('click', () => {
    const isHidden = helpPanel.classList.toggle('hidden');
    localStorage.setItem('cozy-canvas-help-visible', (!isHidden).toString());
    if (!isHidden) {
      settingsPanel.classList.add('hidden');
    }
  });

  btnCloseHelp.addEventListener('click', () => {
    helpPanel.classList.add('hidden');
    localStorage.setItem('cozy-canvas-help-visible', 'false');
  });

  btnToggleSettings.addEventListener('click', () => {
    const isHidden = settingsPanel.classList.toggle('hidden');
    if (!isHidden) {
      helpPanel.classList.add('hidden');
    }
  });

  btnCloseSettings.addEventListener('click', () => {
    settingsPanel.classList.add('hidden');
  });

  settingPhysics.addEventListener('change', (e) => {
    physicsEnabled = e.target.checked;
    localStorage.setItem('cozy-canvas-setting-physics', physicsEnabled);
    showToast(physicsEnabled ? '🌸 Physics enabled' : '🌸 Physics disabled');
    if (physicsEnabled) startPhysicsSimulation();
    else stopPhysicsSimulation();
  });

  settingSpring.addEventListener('input', (e) => {
    const val = parseInt(e.target.value, 10);
    springStiffness = val / 100;
    springValueDisplay.textContent = springStiffness.toFixed(2);
    localStorage.setItem('cozy-canvas-setting-spring', val);

    if (physicsEnabled && simulation) {
      const linkForce = simulation.force('link');
      if (linkForce) {
        linkForce.strength(springStiffness);
        simulation.alpha(0.3).restart();
      }
    }
  });

  window.addEventListener('contextmenu', (e) => {
    if (isLinkingActive || e.target.closest('.cozy-note') || e.target.id === 'connections-svg') {
      e.preventDefault();
    }
  });

  // FIXED: Logout button fully clears sessionStorage tokens
  btnLogin.addEventListener('click', () => {
    if (currentUser) {
      if (confirm(`Sign out of profile ${currentUser}?`)) {
        sessionStorage.removeItem('cozy-canvas-token');
        sessionStorage.removeItem('cozy-canvas-user');
        authToken = null;
        currentUser = null;
        updateLoginUI();
        showToast('🌸 Logged out. Session token destroyed.');
      }
    } else {
      authMode = 'login';
      syncAuthModeUI();
      loginModal.classList.remove('hidden');
      loginEmailInput.value = '';
      loginPasswordInput.value = '';
      loginUsernameInput.value = '';
      loginWord1InputReal.value = '';
      loginWord2InputReal.value = '';
      setTimeout(() => loginEmailInput.focus(), 100);
    }
  });

  btnCloseLogin.addEventListener('click', () => {
    loginModal.classList.add('hidden');
  });

  loginModal.addEventListener('mousedown', (e) => {
    if (e.target === loginModal) {
      loginModal.classList.add('hidden');
    }
  });

  linkSwitchAuth.addEventListener('click', () => {
    authMode = authMode === 'login' ? 'register' : 'login';
    syncAuthModeUI();
  });

  // FIXED: Local password DB removed. Auth delegated to backend. 
  // Network errors are handled gracefully.
  loginForm.addEventListener('submit', async (e) => {
    e.preventDefault();

    const email = loginEmailInput.value.trim().toLowerCase();
    const password = loginPasswordInput.value;

    if (authMode === 'login') {
      try {
        const res = await apiFetch('/api/auth/login', 'POST', { email, password });
        if (res && res.status === 'success' && res.token) {
          sessionStorage.setItem('cozy-canvas-token', res.token);
          sessionStorage.setItem('cozy-canvas-user', res.username);
          authToken = res.token;
          currentUser = res.username;
          
          updateLoginUI();
          loginModal.classList.add('hidden');
          showToast(`🌸 Welcome back, ${currentUser}! Session active.`);
          await init();
        } else {
          showToast('❌ Invalid Email or Password!');
        }
      } catch (err) {
        showToast('❌ Auth server unreachable. Check your connection.');
      }
    } else {
      const username = loginUsernameInput.value.trim();
      const word1 = loginWord1InputReal.value.trim();
      const word2 = loginWord2InputReal.value.trim();

      if (!word1 || !word2) {
        showToast('❌ Please provide both code words!');
        return;
      }

      try {
        const res = await apiFetch('/api/auth/register', 'POST', {
          username,
          email,
          password,
          codewords: [word1, word2]
        });
        if (res && res.status === 'success') {
          showToast(`🌸 Profile ${username} created successfully! You can now log in.`);
          authMode = 'login';
          syncAuthModeUI();
        } else {
          showToast(`❌ Error: ${res ? res.message : 'Unknown failure'}`);
        }
      } catch (err) {
        showToast('❌ Failed to send registration request.');
      }
    }
  });

  window.addEventListener('resize', () => {
    updateTransform();
  });
}

function syncAuthModeUI() {
  if (authMode === 'login') {
    registerFields.classList.add('hidden');
    loginUsernameInput.removeAttribute('required');
    loginWord1InputReal.removeAttribute('required');
    loginWord2InputReal.removeAttribute('required');
    modalTitle.textContent = 'Login to Cozy Cluster';
    modalSubtitle.textContent = 'Connect your engineer profile';
    authSubmitBtn.textContent = 'Login ✨';

    const promptTextNode = document.querySelector('.modal-switch-text').childNodes[0];
    if (promptTextNode) promptTextNode.textContent = "Don't have an account yet? ";
    linkSwitchAuth.textContent = 'Sign Up';
  } else {
    registerFields.classList.remove('hidden');
    loginUsernameInput.setAttribute('required', 'true');
    loginWord1InputReal.setAttribute('required', 'true');
    loginWord2InputReal.setAttribute('required', 'true');
    modalTitle.textContent = 'Sign Up for Cozy Cluster';
    modalSubtitle.textContent = 'Create your engineer profile in one click';
    authSubmitBtn.textContent = 'Create Account ✨';

    const promptTextNode = document.querySelector('.modal-switch-text').childNodes[0];
    if (promptTextNode) promptTextNode.textContent = 'Already have an account? ';
    linkSwitchAuth.textContent = 'Login';
  }
}

function animateCursor() {
  cursorX += (targetMouseX - cursorX) * cursorLerpFactor;
  cursorY += (targetMouseY - cursorY) * cursorLerpFactor;

  cursorEl.style.transform = `translate(${cursorX}px, ${cursorY}px) translate(-50%, -50%)`;

  requestAnimationFrame(animateCursor);
}

function updateStatus() {
  const activeList = currentMode === 'notes' ? notes : envNotes;
  notesCountDisplay.textContent = activeList.length.toString();

  const notesCountIcon = document.querySelector('.waybar-stat:last-child .stat-icon');
  if (notesCountIcon) {
    notesCountIcon.textContent = currentMode === 'notes' ? '📝' : '⚙️';
  }
}

// ==========================================================================
// D3 Physics
// ==========================================================================

function startPhysicsSimulation() {
  if (!physicsEnabled) return;
  if (simulation) {
    simulation.stop();
  }

  const activeNotesList = currentMode === 'notes' ? notes : envNotes;

  simulation = d3.forceSimulation(activeNotesList)
    .force('link', d3.forceLink()
      .id(d => d.id)
      .distance(280)
      .strength(currentMode === 'notes' ? springStiffness : 0.05)
    )
    .force('collide', d3.forceCollide()
      .radius(105)
      .strength(0.8)
    )
    .force('charge', d3.forceManyBody()
      .strength(-150)
      .distanceMax(600)
    )
    .force('center-gravity', d3.forceX(window.innerWidth / 2 - 85).strength(0.01))
    .force('center-gravity-y', d3.forceY(window.innerHeight / 2 - 85).strength(0.01));

  updatePhysicsSimulationLinks();

  simulation.on('tick', () => {
    activeNotesList.forEach(note => {
      const noteEl = document.getElementById(`note-${note.id}`);
      if (noteEl) {
        noteEl.style.left = `${note.x}px`;
        noteEl.style.top = `${note.y}px`;
      }
    });
    updateConnections();
  });

  simulation.alpha(1).restart();
}

function updatePhysicsSimulationLinks() {
  if (!simulation) return;
  const linkForce = simulation.force('link');
  if (!linkForce) return;

  if (currentMode === 'notes') {
    linkForce.links(connections);
  } else {
    const envLinks = [];
    const maxDistance = 380;
    for (let i = 0; i < envNotes.length; i++) {
      for (let j = i + 1; j < envNotes.length; j++) {
        const dx = envNotes[i].x - envNotes[j].x;
        const dy = envNotes[i].y - envNotes[j].y;
        const dist = Math.sqrt(dx * dx + dy * dy);
        if (dist < maxDistance) {
          envLinks.push({ source: envNotes[i].id, target: envNotes[j].id });
        }
      }
    }
    linkForce.links(envLinks);
  }
}

function stopPhysicsSimulation() {
  if (simulation) {
    simulation.stop();
    simulation = null;
  }
}

// ==========================================================================
// Link Mode Handlers
// ==========================================================================

function startLinking(sourceId, clientX, clientY) {
  if (currentMode !== 'notes') return;

  isLinkingActive = true;
  linkSourceNoteId = sourceId;

  const sourceEl = document.getElementById(`note-${sourceId}`);
  if (sourceEl) {
    sourceEl.classList.add('link-source');
  }

  const linkIndicator = document.getElementById('link-indicator');
  if (linkIndicator) {
    linkIndicator.classList.remove('hidden');
  }

  linkPreviewLine = document.createElementNS('http://www.w3.org/2000/svg', 'line');
  linkPreviewLine.classList.add('connection-line-preview');
  connectionsSvg.appendChild(linkPreviewLine);

  updateLinkPreview(clientX, clientY);
}

function updateLinkPreview(clientX, clientY) {
  if (!isLinkingActive || !linkSourceNoteId || !linkPreviewLine) return;

  const sourceNote = notes.find(n => n.id === linkSourceNoteId);
  if (!sourceNote) return;

  const rOffset = 85;
  const startX = sourceNote.x + rOffset;
  const startY = sourceNote.y + rOffset;

  const mouseCanvas = screenToCanvas(clientX, clientY);

  linkPreviewLine.setAttribute('x1', startX + 250000);
  linkPreviewLine.setAttribute('y1', startY + 250000);
  linkPreviewLine.setAttribute('x2', mouseCanvas.x + 250000);
  linkPreviewLine.setAttribute('y2', mouseCanvas.y + 250000);
}

function finishLinking(clientX, clientY) {
  isLinkingActive = false;

  const sourceEl = document.getElementById(`note-${linkSourceNoteId}`);
  if (sourceEl) {
    sourceEl.classList.remove('link-source');
  }

  const linkIndicator = document.getElementById('link-indicator');
  if (linkIndicator) {
    linkIndicator.classList.add('hidden');
  }

  if (linkPreviewLine && linkPreviewLine.parentNode) {
    linkPreviewLine.parentNode.removeChild(linkPreviewLine);
  }
  linkPreviewLine = null;

  const targetEl = document.querySelector('.cozy-note.link-target');
  if (targetEl) {
    const targetId = targetEl.id.replace('note-', '');
    createConnection(linkSourceNoteId, targetId);
    targetEl.classList.remove('link-target');
  }

  document.querySelectorAll('.cozy-note').forEach(el => {
    el.classList.remove('link-target');
  });

  linkSourceNoteId = null;
}

window.addEventListener('DOMContentLoaded', init);