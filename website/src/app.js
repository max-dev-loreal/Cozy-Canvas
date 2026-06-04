// Cozy Canvas - Core Application Logic

import { visOptions } from './vis-config.js';
import * as api from './api.js';
import * as ui from './ui.js';

// Application State
let notes = [];
let connections = [];
let currentUser = null;
let authToken = null;
let network = null;
let nodesDataSet = null;
let edgesDataSet = null;
let currentNoteId = null;
let currentMode = 'notes'; // 'notes' or 'env'

// RMB Linking State
let isLinkingActive = false;
let linkSourceNoteId = null;
let linkCurrentMousePos = null;

// Debounce timer for auto-save on drag
let syncTimeout = null;

export async function init() {
  // Read tokens from sessionStorage
  authToken = sessionStorage.getItem('cozy-canvas-token') || null;
  currentUser = sessionStorage.getItem('cozy-canvas-user') || null;
  
  // Initialize panels toggle listeners
  setupPanelListeners();

  // Update UI waybar state
  ui.updateWaybar(currentUser, notes.length);

  // Setup custom API hooks
  window.addEventListener('cozy-unauthorized', () => {
    ui.showToast('Authentication expired. Please sign in again.', 'error');
    authToken = null;
    currentUser = null;
    ui.updateWaybar(null, 0);
    ui.elements.authModal.classList.remove('hidden');
  });

  window.addEventListener('cozy-network-error', () => {
    ui.showToast('Server unreachable. Offline mode.', 'error');
  });

  // Check auth
  if (authToken) {
    await loadData();
  } else {
    // Show login modal
    ui.elements.authModal.classList.remove('hidden');
    ui.toggleAuthMode('login');
  }

  // Global listeners
  setupGlobalListeners();
  setupAuthFormListeners();
}

function setupPanelListeners() {
  // Settings toggles
  ui.elements.btnToggleSettings.addEventListener('click', () => {
    ui.elements.settingsPanel.classList.toggle('hidden');
    ui.elements.helpPanel.classList.add('hidden');
  });
  
  ui.elements.btnCloseSettings.addEventListener('click', () => {
    ui.elements.settingsPanel.classList.add('hidden');
  });

  // Help toggles
  ui.elements.btnToggleHelp.addEventListener('click', () => {
    ui.elements.helpPanel.classList.toggle('hidden');
    ui.elements.settingsPanel.classList.add('hidden');
  });
  
  ui.elements.btnCloseHelp.addEventListener('click', () => {
    ui.elements.helpPanel.classList.add('hidden');
  });

  // Mode switcher
  ui.elements.modeNotes.addEventListener('click', async () => {
    if (currentMode === 'notes') return;
    currentMode = 'notes';
    ui.elements.modeNotes.classList.add('active');
    ui.elements.modeEnv.classList.remove('active');
    await loadData();
  });

  ui.elements.modeEnv.addEventListener('click', async () => {
    if (currentMode === 'env') return;
    currentMode = 'env';
    ui.elements.modeNotes.classList.remove('active');
    ui.elements.modeEnv.classList.add('active');
    await loadData();
  });

  // Login / Logout button
  ui.elements.btnLogin.addEventListener('click', () => {
    if (authToken) {
      // Logout
      sessionStorage.removeItem('cozy-canvas-token');
      sessionStorage.removeItem('cozy-canvas-user');
      authToken = null;
      currentUser = null;
      ui.showToast('Signed out successfully.');
      ui.updateWaybar(null, 0);
      
      // Clean graph
      if (network) {
        network.destroy();
        network = null;
      }
      notes = [];
      connections = [];
      ui.elements.authModal.classList.remove('hidden');
      ui.toggleAuthMode('login');
    } else {
      ui.elements.authModal.classList.remove('hidden');
      ui.toggleAuthMode('login');
    }
  });

  // Physics settings listeners
  ui.elements.settingPhysics.addEventListener('change', (e) => {
    if (network) {
      network.setOptions({ physics: { enabled: e.target.checked } });
    }
  });

  ui.elements.settingSpring.addEventListener('input', (e) => {
    const val = parseFloat(e.target.value) / 100;
    ui.elements.springValue.textContent = val.toFixed(2);
    if (network) {
      network.setOptions({
        physics: {
          barnesHut: {
            springConstant: val
          }
        }
      });
    }
  });

  // Close modals clicking outside
  window.addEventListener('click', (e) => {
    if (e.target === ui.elements.authModal) {
      ui.elements.authModal.classList.add('hidden');
    }
    if (e.target === ui.elements.noteModal) {
      ui.closeNoteModal();
    }
  });

  ui.elements.btnCloseAuthModal.addEventListener('click', () => {
    ui.elements.authModal.classList.add('hidden');
  });

  ui.elements.btnCloseNoteModal.addEventListener('click', () => {
    ui.closeNoteModal();
  });
}

function setupGlobalListeners() {
  // Ctrl+S handler
  window.addEventListener('keydown', (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      e.preventDefault();
      syncCurrentData();
    }
    if (e.key === 'Escape') {
      ui.closeNoteModal();
      ui.elements.authModal.classList.add('hidden');
      ui.elements.settingsPanel.classList.add('hidden');
      ui.elements.helpPanel.classList.add('hidden');
    }
  });
}

let activeAuthMode = 'login';
function setupAuthFormListeners() {
  // Switch links
  ui.elements.authSwitchLink.addEventListener('click', () => {
    if (activeAuthMode === 'login') {
      activeAuthMode = 'register';
      ui.toggleAuthMode('register');
    } else {
      activeAuthMode = 'login';
      ui.toggleAuthMode('login');
    }
  });

  // Submit form
  ui.elements.authForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const email = ui.elements.authEmail.value;
    const password = ui.elements.authPassword.value;

    try {
      if (activeAuthMode === 'login') {
        const res = await api.login({ email, password });
        if (res && res.token) {
          authToken = res.token;
          currentUser = res.username;
          ui.elements.authModal.classList.add('hidden');
          ui.updateWaybar(currentUser, 0);
          ui.showToast(`Welcome back, ${currentUser}!`);
          await loadData();
        }
      } else {
        const username = ui.elements.authUsername.value;
        const w1 = ui.elements.authCodeword1.value;
        const w2 = ui.elements.authCodeword2.value;

        const res = await api.register({
          username,
          email,
          password,
          codewords: [w1, w2]
        });

        if (res && res.status === 'success') {
          ui.showToast('Registration successful! Please login.');
          activeAuthMode = 'login';
          ui.toggleAuthMode('login');
        }
      }
    } catch (err) {
      console.error(err);
      ui.showToast(err.message || 'Authorization failed', 'error');
    }
  });
}

export async function loadData() {
  try {
    if (currentMode === 'notes') {
      const notesRes = await api.getNotes();
      const connRes = await api.getConnections();
      
      notes = notesRes.notes || notesRes || [];
      connections = connRes.connections || connRes || [];
    } else {
      const envRes = await api.getEnvNotes();
      notes = envRes.notes || envRes || [];
      connections = [];
    }
    
    ui.updateWaybar(currentUser, notes.length);
    renderGraph();
  } catch (err) {
    console.error('Failed to load data:', err);
    ui.showToast('Failed to load canvas data from cloud', 'error');
  }
}

function renderGraph() {
  const container = document.getElementById('graph-canvas');
  if (!container) return;

  // Format notes dataset
  const nodesArray = notes.map(note => ({
    id: note.id,
    label: note.title || 'New Note',
    x: note.x,
    y: note.y
  }));

  // Format connections dataset
  const edgesArray = connections.map(conn => ({
    id: `${conn.source_note_id}-${conn.target_note_id}`,
    from: conn.source_note_id,
    to: conn.target_note_id
  }));

  nodesDataSet = new vis.DataSet(nodesArray);
  edgesDataSet = new vis.DataSet(edgesArray);

  // Set physics state from current setting checkbox
  visOptions.physics.enabled = ui.elements.settingPhysics.checked;
  const currentSpringVal = parseFloat(ui.elements.settingSpring.value) / 100;
  visOptions.physics.barnesHut.springConstant = currentSpringVal;

  network = new vis.Network(container, { nodes: nodesDataSet, edges: edgesDataSet }, visOptions);

  // Redraw zoom stats
  document.getElementById('zoom-display').textContent = `${Math.round(network.getScale() * 100)}%`;

  // Attach Vis.js listeners
  setupNetworkListeners();
}

function setupNetworkListeners() {
  const container = document.getElementById('graph-canvas');
  
  // Track cursor position and convert to canvas coordinates
  container.addEventListener('mousemove', (e) => {
    if (!network) return;
    const canvasPos = network.DOMtoCanvas({ x: e.offsetX, y: e.offsetY });
    ui.elements.coordDisplay.textContent = `${Math.round(canvasPos.x)}, ${Math.round(canvasPos.y)}`;
    
    // If linking active, redraw to render preview line
    if (isLinkingActive) {
      linkCurrentMousePos = canvasPos;
      network.redraw();
    }
  });

  // Track right-click linking start
  container.addEventListener('mousedown', (e) => {
    if (e.button === 2 && network) { // Right Click
      const clickPos = { x: e.offsetX, y: e.offsetY };
      const nodeId = network.getNodeAt(clickPos);
      if (nodeId) {
        e.stopPropagation();
        e.preventDefault();
        
        isLinkingActive = true;
        linkSourceNoteId = nodeId;
        linkCurrentMousePos = network.DOMtoCanvas(clickPos);
        network.redraw();
      }
    }
  });

  // Track right-click linking release
  container.addEventListener('mouseup', (e) => {
    if (e.button === 2 && isLinkingActive && network) {
      e.stopPropagation();
      e.preventDefault();

      const clickPos = { x: e.offsetX, y: e.offsetY };
      const targetId = network.getNodeAt(clickPos);

      if (targetId && targetId !== linkSourceNoteId) {
        // Prevent duplicate connections
        const exists = connections.some(c => 
          (c.source_note_id === linkSourceNoteId && c.target_note_id === targetId) ||
          (c.source_note_id === targetId && c.target_note_id === linkSourceNoteId)
        );

        if (!exists) {
          const newConn = {
            source_note_id: linkSourceNoteId,
            target_note_id: targetId,
            created_at: new Date().toISOString()
          };
          connections.push(newConn);
          edgesDataSet.add({
            id: `${linkSourceNoteId}-${targetId}`,
            from: linkSourceNoteId,
            to: targetId
          });
          syncCurrentData();
          ui.showToast('Connection linked! 🔗');
        }
      }
      
      isLinkingActive = false;
      linkSourceNoteId = null;
      linkCurrentMousePos = null;
      network.redraw();
    }
  });

  // Prevent default context menus
  container.addEventListener('contextmenu', (e) => {
    e.preventDefault();
  });

  // Drawing listener for linking line previews
  network.on('afterDrawing', (ctx) => {
    if (isLinkingActive && linkSourceNoteId && linkCurrentMousePos) {
      const sourcePos = network.getPositions([linkSourceNoteId])[linkSourceNoteId];
      if (sourcePos) {
        ctx.save();
        ctx.strokeStyle = 'rgba(232, 160, 191, 0.6)';
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.moveTo(sourcePos.x, sourcePos.y);
        ctx.lineTo(linkCurrentMousePos.x, linkCurrentMousePos.y);
        ctx.stroke();
        ctx.restore();
      }
    }
  });

  // Node clicks
  network.on('click', (params) => {
    if (params.nodes.length > 0) {
      const nodeId = params.nodes[0];
      openNoteModal(nodeId);
    }
  });

  // Double clicks on background or nodes
  network.on('doubleClick', (params) => {
    if (params.nodes.length === 0) {
      const canvasPos = params.pointer.canvas;
      createNewNoteAt(canvasPos.x, canvasPos.y);
    }
  });

  // Drag node coordinate persistence
  network.on('dragEnd', (params) => {
    if (params.nodes.length > 0) {
      const nodeId = params.nodes[0];
      const newPos = network.getPositions([nodeId])[nodeId];
      if (newPos) {
        const note = notes.find(n => n.id === nodeId);
        if (note) {
          note.x = newPos.x;
          note.y = newPos.y;
        }
        
        // Debounce coordinate sync
        if (syncTimeout) clearTimeout(syncTimeout);
        syncTimeout = setTimeout(() => {
          syncCurrentData();
        }, 500);
      }
    }
  });

  // Zoom changes
  network.on('zoom', () => {
    const scale = network.getScale();
    ui.elements.zoomDisplay.textContent = `${Math.round(scale * 100)}%`;
  });

  // Node Tooltips on hover
  network.on('hoverNode', (params) => {
    const nodeId = params.node;
    const note = notes.find(n => n.id === nodeId);
    if (note && note.context) {
      ui.elements.nodeTooltip.textContent = note.context;
      ui.elements.nodeTooltip.classList.add('visible');
    }
  });

  network.on('blurNode', () => {
    ui.elements.nodeTooltip.classList.remove('visible');
  });
}

function createNewNoteAt(x, y) {
  const id = 'note-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
  const note = {
    id,
    title: 'New Note',
    text: '',
    context: '',
    x,
    y,
    created_at: new Date().toISOString()
  };
  
  notes.push(note);
  nodesDataSet.add({
    id,
    label: note.title,
    x,
    y
  });
  
  ui.elements.notesCount.textContent = notes.length;
  syncCurrentData();
}

function openNoteModal(noteId) {
  currentNoteId = noteId;
  const note = notes.find(n => n.id === noteId);
  if (!note) return;

  ui.openNoteModal(note, saveNoteFromModal, deleteNote);
}

function saveNoteFromModal(updatedNote) {
  const note = notes.find(n => n.id === currentNoteId);
  if (note) {
    note.title = updatedNote.title;
    note.context = updatedNote.context;
    note.text = updatedNote.text;
    
    // Update node title label in Vis.js
    nodesDataSet.update({
      id: currentNoteId,
      label: note.title
    });
    
    syncCurrentData();
  }
}

async function syncCurrentData() {
  if (!authToken) return;
  try {
    await api.sync(notes, connections);
    ui.showToast('Saved to cloud');
  } catch (err) {
    console.error('Failed to sync graph:', err);
    ui.showToast('Failed to save to cloud', 'error');
  }
}

function deleteNote(noteId) {
  // Clean notes list
  notes = notes.filter(n => n.id !== noteId);
  
  // Clean connections list
  connections = connections.filter(c => c.source_note_id !== noteId && c.target_note_id !== noteId);
  
  // Delete from Vis.js DataSet
  nodesDataSet.remove(noteId);
  
  // Vis.js automatically removes edge components related to deleted node in view,
  // but we should manually clean our internal edgesDataSet too.
  const edgesToRemove = edgesDataSet.get({
    filter: (edge) => edge.from === noteId || edge.to === noteId
  });
  edgesDataSet.remove(edgesToRemove);
  
  // Hide active tooltips if the deleted node was hovered
  ui.elements.nodeTooltip.classList.remove('visible');

  ui.elements.notesCount.textContent = notes.length;
  
  syncCurrentData();

  // If active modal, hide it
  if (currentNoteId === noteId) {
    ui.closeNoteModal();
    currentNoteId = null;
  }
}
