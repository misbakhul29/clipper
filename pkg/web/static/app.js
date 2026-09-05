// Clipper Studio — Minimalist Video Studio Application Engine
let player = document.getElementById('videoPlayer') || document.getElementById('mainPlayer');
let segments = [];
let pollTimer = null;

// Theme Switcher (Light / Dark mode)
function initTheme() {
  const saved = localStorage.getItem('clipper-theme') || 'dark';
  applyTheme(saved);
}

function toggleTheme() {
  const current = document.documentElement.getAttribute('data-theme') || 'dark';
  const next = current === 'dark' ? 'light' : 'dark';
  document.documentElement.classList.add('theme-transition');
  applyTheme(next);
  setTimeout(() => {
    document.documentElement.classList.remove('theme-transition');
  }, 250);
}

function applyTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme);
  localStorage.setItem('clipper-theme', theme);

  const icon = document.getElementById('themeIcon');
  if (!icon) return;

  if (theme === 'dark') {
    // Show Sun icon (switches to light)
    icon.innerHTML = `
      <circle cx="12" cy="12" r="5"/>
      <line x1="12" y1="1" x2="12" y2="3"/>
      <line x1="12" y1="21" x2="12" y2="23"/>
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
      <line x1="1" y1="12" x2="3" y2="12"/>
      <line x1="21" y1="12" x2="23" y2="12"/>
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
    `;
  } else {
    // Show Moon icon (switches to dark)
    icon.innerHTML = `
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
    `;
  }
}

// Player time update and lifecycle events handler
function setupPlayerEvents() {
  if (!player) {
    player = document.getElementById('videoPlayer') || document.getElementById('mainPlayer');
  }
  if (player) {
    player.addEventListener('timeupdate', () => {
      const cur = player.currentTime || 0;
      const dur = player.duration || 0;
      const el = document.getElementById('timeDisplay') || document.getElementById('playheadTime');
      if (el) {
        el.textContent = formatTimecode(cur) + ' / ' + formatTimecode(dur);
      }
    });

    player.addEventListener('loadedmetadata', () => {
      const ph = document.getElementById('playerPlaceholder');
      if (ph) ph.style.display = 'none';
      const errEl = document.getElementById('playerError');
      if (errEl) errEl.style.display = 'none';
      const loader = document.getElementById('playerLoader');
      if (loader) loader.style.display = 'none';

      const cur = player.currentTime || 0;
      const dur = player.duration || 0;
      const el = document.getElementById('timeDisplay') || document.getElementById('playheadTime');
      if (el) {
        el.textContent = formatTimecode(cur) + ' / ' + formatTimecode(dur);
      }

      const endEl = document.getElementById('trimEnd');
      if (endEl && (endEl.value === '00:30' || endEl.value === '00:00' || !endEl.value)) {
        endEl.value = formatSimple(Math.min(dur, 30));
      }
    });

    player.addEventListener('canplay', () => {
      const loader = document.getElementById('playerLoader');
      if (loader) loader.style.display = 'none';
      const ph = document.getElementById('playerPlaceholder');
      if (ph) ph.style.display = 'none';
      const errEl = document.getElementById('playerError');
      if (errEl) errEl.style.display = 'none';
    });

    player.addEventListener('play', () => {
      const ph = document.getElementById('playerPlaceholder');
      if (ph) ph.style.display = 'none';
      const errEl = document.getElementById('playerError');
      if (errEl) errEl.style.display = 'none';
    });

    player.addEventListener('error', (e) => {
      const loader = document.getElementById('playerLoader');
      if (loader) loader.style.display = 'none';
      const ph = document.getElementById('playerPlaceholder');
      if (ph) ph.style.display = 'none';

      const errEl = document.getElementById('playerError');
      const errSub = document.getElementById('playerErrorSub');
      const errTitle = document.getElementById('playerErrorTitle');

      let msg = 'Format video tidak didukung atau video gagal dimuat.';
      if (player.error) {
        switch (player.error.code) {
          case 1: // MEDIA_ERR_ABORTED
            msg = 'Pemuatan video dibatalkan oleh pengguna.';
            break;
          case 2: // MEDIA_ERR_NETWORK
            msg = 'Kesalahan jaringan saat mengunduh/streaming video.';
            break;
          case 3: // MEDIA_ERR_DECODE
            msg = 'Gagal melakukan decode video. Codec/kompresi tidak didukung browser.';
            break;
          case 4: // MEDIA_ERR_SRC_NOT_SUPPORTED
            msg = 'Format video tidak didukung oleh browser atau file tidak ditemukan (404). Gunakan format MP4 (H.264/AAC) atau WebM.';
            break;
          default:
            msg = player.error.message || 'Terjadi kesalahan pada pemutaran media.';
        }
      }

      if (errTitle) errTitle.textContent = 'Gagal Memutar Video';
      if (errSub) errSub.textContent = msg;
      if (errEl) errEl.style.display = 'flex';
    });
  }
}

function formatTimecode(sec) {
  if (isNaN(sec) || sec < 0) return '00:00.00';
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  const cs = Math.floor((sec % 1) * 100);
  const pad = (n) => (n < 10 ? '0' + n : n);
  return `${pad(m)}:${pad(s)}.${pad(cs)}`;
}

function formatSimple(sec) {
  if (isNaN(sec) || sec < 0) return '00:00';
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  const pad = (n) => (n < 10 ? '0' + n : n);
  return `${pad(m)}:${pad(s)}`;
}

function parseSeconds(str) {
  if (!str) return 0;
  const parts = str.trim().split(':');
  if (parts.length === 2) {
    return parseFloat(parts[0]) * 60 + parseFloat(parts[1]);
  } else if (parts.length === 3) {
    return parseFloat(parts[0]) * 3600 + parseFloat(parts[1]) * 60 + parseFloat(parts[2]);
  }
  return parseFloat(str) || 0;
}

function seekRelative(delta) {
  if (!player) player = document.getElementById('videoPlayer');
  if (player && !isNaN(player.duration)) {
    player.currentTime = Math.max(0, Math.min(player.duration, player.currentTime + delta));
  }
}

function togglePlay() {
  if (!player) player = document.getElementById('videoPlayer');
  if (!player) return;
  if (player.paused) {
    player.play().catch(() => {});
  } else {
    player.pause();
  }
}

function markCurrent(target) {
  if (!player) player = document.getElementById('videoPlayer');
  if (!player) return;
  const cur = player.currentTime || 0;
  const formatted = formatSimple(cur);
  if (target === 'start') {
    document.getElementById('trimStart').value = formatted;
  } else {
    document.getElementById('trimEnd').value = formatted;
  }
}

function addSegment() {
  const startEl = document.getElementById('trimStart');
  const endEl = document.getElementById('trimEnd');
  const titleEl = document.getElementById('trimTitle');

  const start = startEl.value.trim();
  const end = endEl.value.trim();
  const title = titleEl.value.trim();

  if (!start || !end) {
    alert('Please enter start and end timestamps.');
    return;
  }

  const startSec = parseSeconds(start);
  const endSec = parseSeconds(end);
  if (endSec <= startSec) {
    alert('End timestamp must be greater than start timestamp.');
    return;
  }

  segments.push({ start, end, title });
  titleEl.value = '';
  renderSegments();
}

function removeSegment(idx) {
  segments.splice(idx, 1);
  renderSegments();
}

function seekToSegment(startStr) {
  if (!player) player = document.getElementById('videoPlayer');
  if (!player) return;
  player.currentTime = parseSeconds(startStr);
  player.play().catch(() => {});
}

function renderSegments() {
  const list = document.getElementById('segmentStack');
  const countEl = document.getElementById('queueCount');
  if (countEl) countEl.textContent = segments.length;

  if (!list) return;

  if (segments.length === 0) {
    list.innerHTML = `
      <div class="empty-state">
        <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" style="opacity:0.6;"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 14 14"/></svg>
        <span>No segments added yet. Use the trimmer or AI detection on the left.</span>
      </div>`;
    return;
  }

  list.innerHTML = segments.map((s, i) => {
    const cueCount = (s.subtitles && s.subtitles.length > 0) ? `${s.subtitles.length} cues` : 'Subtitles';
    const cueBtnClass = (s.subtitles && s.subtitles.length > 0) ? 'btn-primary' : 'btn-secondary';
    return `
    <div class="segment-item">
      <div>
        <div class="segment-title">${s.title ? escapeHtml(s.title) : 'Segment #' + (i + 1)}</div>
        <div class="segment-range">${s.start} &rarr; ${s.end}</div>
      </div>
      <div style="display:flex; gap:6px; align-items:center;">
        <button class="btn ${cueBtnClass} segment-btn-sub" onclick="openSubtitleStudio(${i})" title="Configure Subtitles">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:12px; height:12px;"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
          <span>${cueCount}</span>
        </button>
        <button class="btn btn-secondary btn-sm" onclick="seekToSegment('${s.start}')" title="Play Segment">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
        </button>
        <button class="btn btn-secondary btn-sm" onclick="removeSegment(${i})" title="Remove">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>
    </div>
  `;}).join('');
}

function onAutoDetectModeChange() {
  const mode = document.getElementById('autoDetectMode').value;
  const btnText = document.getElementById('autoDetectBtnText');
  if (mode === 'ai') {
    if (btnText) btnText.textContent = 'Generate with AI';
  } else if (mode === 'silence') {
    if (btnText) btnText.textContent = 'Detect Silence';
  } else if (mode === 'scene') {
    if (btnText) btnText.textContent = 'Detect Scenes';
  }
}

async function runAutoDetect() {
  const src = document.getElementById('videoSource').value.trim();
  if (!src) {
    alert('Please enter or load a video source first.');
    return;
  }

  const mode = document.getElementById('autoDetectMode').value;
  const btn = document.getElementById('autoDetectBtn');
  const btnText = document.getElementById('autoDetectBtnText');
  const loader = document.getElementById('playerLoader');
  const title = document.getElementById('loaderTitle');
  const sub = document.getElementById('loaderSub');

  if (btn) btn.disabled = true;
  if (btnText) btnText.textContent = 'Analyzing...';
  if (loader) loader.style.display = 'flex';

  if (mode === 'ai') {
    if (title) title.textContent = 'AI Highlight Analysis...';
    if (sub) sub.textContent = 'Fetching transcript and analyzing viral highlights with AI.';
  } else if (mode === 'silence') {
    if (title) title.textContent = 'Detecting Speech Segments...';
    if (sub) sub.textContent = 'Scanning audio stream for voice & dialogue boundaries.';
  } else {
    if (title) title.textContent = 'Detecting Scene Changes...';
    if (sub) sub.textContent = 'Scanning video stream for visual camera shot cuts.';
  }

  const durVal = parseFloat(document.getElementById('autoDetectDuration')?.value || '0');
  const payload = {
    input_file: src,
    mode: mode,
    ai_router: 'gemini',
    model: getGlobalSegmentModel(),
    api_key: getGlobalApiKey(),
    ai_configs: getGlobalAIConfigs(),
    routing_models: getGlobalRoutingModels(),
    shorts: document.getElementById('cfgShorts').checked,
    target_duration: durVal
  };

  try {
    const res = await fetch('/api/auto-detect', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    const data = await res.json();

    if (!res.ok) {
      alert(data.error || 'Failed to auto-detect segments.');
      return;
    }

    if (!data.segments || data.segments.length === 0) {
      alert('No segments were detected. Try another detection mode.');
      return;
    }

    // Append detected segments to queue
    segments = segments.concat(data.segments);
    renderSegments();
    switchTab('queue');
  } catch (err) {
    alert('Network error during auto-detection: ' + err);
  } finally {
    if (btn) btn.disabled = false;
    onAutoDetectModeChange();
    if (loader) loader.style.display = 'none';
  }
}

function switchTab(tab) {
  const queueBtn = document.getElementById('tabQueueBtn');
  const clipsBtn = document.getElementById('tabClipsBtn');
  const tabQueue = document.getElementById('tabQueue');
  const tabClips = document.getElementById('tabClips');

  if (queueBtn) queueBtn.classList.toggle('active', tab === 'queue');
  if (clipsBtn) clipsBtn.classList.toggle('active', tab === 'clips');
  if (tabQueue) tabQueue.style.display = tab === 'queue' ? 'block' : 'none';
  if (tabClips) tabClips.style.display = tab === 'clips' ? 'block' : 'none';

  if (tab === 'clips') {
    loadClipsGallery();
  }
}

function loadSource() {
  loadSourceVideo();
}

function handleFileSelect(event) {
  const file = event.target.files && event.target.files[0];
  if (!file) return;

  const srcInput = document.getElementById('videoSource');
  if (srcInput) {
    srcInput.value = file.name;
  }

  const ph = document.getElementById('playerPlaceholder');
  if (ph) ph.style.display = 'none';
  const errEl = document.getElementById('playerError');
  if (errEl) errEl.style.display = 'none';
  const loader = document.getElementById('playerLoader');
  if (loader) loader.style.display = 'none';

  if (!player) player = document.getElementById('videoPlayer') || document.getElementById('mainPlayer');
  if (player) {
    const blobURL = URL.createObjectURL(file);
    player.src = blobURL;
    player.load();
    player.play().catch(() => {});
  }
}

function retryVideoLoad() {
  const errEl = document.getElementById('playerError');
  if (errEl) errEl.style.display = 'none';
  loadSourceVideo();
}

async function loadSourceVideo() {
  const src = document.getElementById('videoSource').value.trim();
  if (!src) {
    alert('Please enter a local video file path, select Browse, or enter a YouTube URL.');
    return;
  }

  const loader = document.getElementById('playerLoader');
  const title = document.getElementById('loaderTitle');
  const sub = document.getElementById('loaderSub');
  const loadBtn = document.getElementById('loadBtn');
  const loadBtnText = document.getElementById('loadBtnText');
  const ph = document.getElementById('playerPlaceholder');
  const errEl = document.getElementById('playerError');

  if (errEl) errEl.style.display = 'none';

  // Show loading state
  if (loader) loader.style.display = 'flex';
  if (loadBtn) loadBtn.disabled = true;
  if (loadBtnText) loadBtnText.textContent = 'Loading...';

  const isYT = src.startsWith('http://') || src.startsWith('https://') || src.includes('youtube.com') || src.includes('youtu.be');
  if (isYT) {
    if (title) title.textContent = 'Downloading YouTube video...';
    if (sub) sub.textContent = 'Fetching video into local cache for frame-accurate player scrubbing and clipping.';
  } else {
    if (title) title.textContent = 'Preparing video...';
    if (sub) sub.textContent = 'Loading video file into browser preview player.';
  }

  try {
    const res = await fetch('/api/prepare', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source: src })
    });
    const data = await res.json();

    if (!res.ok) {
      if (loader) loader.style.display = 'none';
      const errSub = document.getElementById('playerErrorSub');
      const errTitle = document.getElementById('playerErrorTitle');
      if (errTitle) errTitle.textContent = 'File / Video Error';
      if (errSub) errSub.textContent = data.error || 'Failed to prepare video.';
      if (errEl) errEl.style.display = 'flex';
      return;
    }

    if (!player) player = document.getElementById('videoPlayer') || document.getElementById('mainPlayer');
    if (player) {
      if (ph) ph.style.display = 'none';
      if (errEl) errEl.style.display = 'none';
      player.src = data.preview_url;
      player.load();
      player.play().catch(() => {});
    } else {
      if (loader) loader.style.display = 'none';
    }
  } catch (err) {
    if (loader) loader.style.display = 'none';
    const errSub = document.getElementById('playerErrorSub');
    const errTitle = document.getElementById('playerErrorTitle');
    if (errTitle) errTitle.textContent = 'Koneksi / Jaringan Error';
    if (errSub) errSub.textContent = 'Error preparing video: ' + err;
    if (errEl) errEl.style.display = 'flex';
  } finally {
    if (loadBtn) loadBtn.disabled = false;
    if (loadBtnText) loadBtnText.textContent = 'Load';
  }
}

function playClipInStudio(url, name) {
  switchTab('queue');
  const srcInput = document.getElementById('videoSource');
  if (srcInput) srcInput.value = url;
  const ph = document.getElementById('playerPlaceholder');
  if (ph) ph.style.display = 'none';
  const errEl = document.getElementById('playerError');
  if (errEl) errEl.style.display = 'none';
  const loader = document.getElementById('playerLoader');
  if (loader) loader.style.display = 'none';

  if (!player) player = document.getElementById('videoPlayer') || document.getElementById('mainPlayer');
  if (player) {
    player.src = url;
    player.load();
    player.play().catch(() => {});
  }
}

async function loadClipsGallery() {
  try {
    const res = await fetch('/api/clips');
    const data = await res.json();
    const grid = document.getElementById('clipsGrid');
    const countEl = document.getElementById('clipsCount');
    const clearAllBtn = document.getElementById('clearAllClipsBtn');
    if (countEl) countEl.textContent = data ? data.length : 0;

    if (clearAllBtn) {
      clearAllBtn.style.display = (!data || data.length === 0) ? 'none' : 'inline-flex';
    }

    if (!grid) return;

    if (!data || data.length === 0) {
      grid.innerHTML = `
        <div class="empty-state">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" style="opacity:0.6;"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>
          <span>No rendered clips found in output directory.</span>
        </div>`;
      return;
    }

    grid.innerHTML = data.map(c => `
      <div class="clip-card">
        ${c.thumbnail_url
          ? `<img src="${c.thumbnail_url}" class="clip-thumb" alt="${escapeHtml(c.name)}" onclick="playClipInStudio('${c.url}', '${escapeHtml(c.name)}')" style="cursor:pointer;" title="Click to preview in Studio" />`
          : `<div class="clip-thumb-empty" onclick="playClipInStudio('${c.url}', '${escapeHtml(c.name)}')" style="cursor:pointer;" title="Click to preview in Studio"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/><line x1="7" y1="2" x2="7" y2="22"/><line x1="17" y1="2" x2="17" y2="22"/><line x1="2" y1="12" x2="22" y2="12"/></svg></div>`
        }
        <div class="clip-details">
          <div class="clip-name" title="${escapeHtml(c.name)}">${escapeHtml(c.name)}</div>
          <div class="clip-meta-row">
            <span>${c.size_str}</span>
            <span>${c.mod_time || ''}</span>
          </div>
          <div class="clip-actions">
            <button onclick="playClipInStudio('${c.url}', '${escapeHtml(c.name)}')" class="btn btn-secondary btn-sm" style="flex:1;" title="Preview in Studio player">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
              Studio
            </button>
            <a href="${c.url}" target="_blank" class="btn btn-secondary btn-sm" title="Open in new tab">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
            </a>
            <a href="${c.url}" download class="btn btn-secondary btn-sm" title="Download">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            </a>
            <button onclick="deleteClip('${escapeHtml(c.name)}')" class="btn btn-secondary btn-sm btn-delete-clip" title="Delete clip file">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
            </button>
          </div>
        </div>
      </div>
    `).join('');
  } catch (err) {
    console.error('Error loading clips:', err);
  }
}

async function deleteClip(name) {
  if (!name) return;
  if (!confirm(`Are you sure you want to delete "${name}"?`)) return;
  try {
    const res = await fetch(`/api/clips?name=${encodeURIComponent(name)}`, {
      method: 'DELETE'
    });
    const data = await res.json();
    if (!res.ok) {
      alert(data.error || 'Failed to delete clip');
      return;
    }
    loadClipsGallery();
    const storageModal = document.getElementById('storageModal');
    if (storageModal && storageModal.style.display !== 'none') {
      fetchStorageStats();
    }
  } catch (err) {
    alert('Error deleting clip: ' + err);
  }
}

function toggleAdvancedRenderOptions() {
  const drawer = document.getElementById('advRenderDrawer');
  const toggleBtn = document.getElementById('advOptionsToggle');
  if (!drawer) return;
  const isHidden = drawer.style.display === 'none';
  drawer.style.display = isHidden ? 'block' : 'none';
  if (toggleBtn) {
    toggleBtn.textContent = isHidden ? '[- Advanced Options]' : '[+ Advanced Options]';
  }
}

function onRenderModeChange() {
  const mode = document.getElementById('cfgMode')?.value;
  if (mode === 'merge') {
    const drawer = document.getElementById('advRenderDrawer');
    if (drawer && drawer.style.display === 'none') {
      toggleAdvancedRenderOptions();
    }
  }
}

async function startClippingJob() {
  const input = document.getElementById('videoSource').value.trim();
  if (!input) {
    alert('Please specify an input video file or YouTube URL.');
    return;
  }
  if (segments.length === 0) {
    alert('Queue is empty. Please add segments manually or click "Generate with AI" / "Detect Silence" first before rendering clips.');
    return;
  }

  const payload = {
    input_file: input,
    segments: segments,
    mode: document.getElementById('cfgMode')?.value || 'split',
    output: document.getElementById('cfgOutputFile')?.value.trim() || 'merged_highlight.mp4',
    strategy: document.getElementById('cfgStrategy')?.value || 'fast',
    quality: document.getElementById('cfgQuality')?.value || 'best',
    shorts: document.getElementById('cfgShorts').checked,
    shorts_style: document.getElementById('cfgShortsStyle').value,
    subtitles: document.getElementById('cfgBurnSubs').checked,
    sub_preset: document.getElementById('cfgSubPreset').value,
    sub_sdh_mode: document.getElementById('cfgSubSDHMode')?.value || 'strip',
    sub_emoji: true,
    loudnorm: document.getElementById('cfgLoudnorm').checked,
    jump_cut: document.getElementById('cfgJumpCut').checked,
    jump_cut_min_silence: parseFloat(document.getElementById('cfgJumpCutMinSil')?.value || '1.0'),
    jump_cut_margin: parseFloat(document.getElementById('cfgJumpCutMargin')?.value || '0.2'),
    jump_cut_noise: parseFloat(document.getElementById('cfgJumpCutNoise')?.value || '-30'),
    watermark: document.getElementById('cfgWatermark')?.value.trim() || '',
    watermark_pos: document.getElementById('cfgWatermarkPos')?.value || 'top-right',
    overlay_text: document.getElementById('cfgOverlayText')?.value.trim() || '',
    text_pos: document.getElementById('cfgTextPos')?.value || 'bottom-center',
    font_color: document.getElementById('cfgFontColor')?.value || 'white',
    generate_metadata: document.getElementById('cfgMetadata').checked,
    extract_thumbnail: document.getElementById('cfgThumbnail').checked,
    thumbnail_count: 1,
    hwaccel: document.getElementById('cfgHwaccel')?.value || 'auto',
    output_dir: document.getElementById('cfgOutputDir')?.value.trim() || './clips',
    ai_configs: getGlobalAIConfigs(),
    routing_models: getGlobalRoutingModels(),
    ai_config: {
      api_key: getGlobalApiKey(),
      segment_model: getGlobalSegmentModel(),
      stt_model: getGlobalSTTModel()
    }
  };

  const card = document.getElementById('progressCard');
  if (card) card.style.display = 'block';

  const statusPill = document.getElementById('statusPill');
  if (statusPill) statusPill.classList.add('rendering');

  try {
    const res = await fetch('/api/render', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    let data = {};
    const text = await res.text();
    try {
      data = JSON.parse(text);
    } catch {
      data = { error: text || `HTTP ${res.status} ${res.statusText}` };
    }

    if (!res.ok) {
      alert(data.error || 'Failed to start rendering.');
      if (card) card.style.display = 'none';
      if (statusPill) statusPill.classList.remove('rendering');
      return;
    }
    startPollingStatus();
  } catch (err) {
    alert('Failed to start rendering: ' + err);
    if (card) card.style.display = 'none';
    if (statusPill) statusPill.classList.remove('rendering');
  }
}

function handleConfigFileImport(event) {
  const file = event.target.files[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = (e) => {
    try {
      const cfg = JSON.parse(e.target.result);
      applyImportedConfig(cfg);
    } catch (err) {
      alert('Failed to parse JSON configuration file: ' + err);
    }
  };
  reader.readAsText(file);
  event.target.value = '';
}

function applyImportedConfig(cfg) {
  if (cfg.input || cfg.input_file) {
    const srcInput = document.getElementById('videoSource');
    if (srcInput) {
      srcInput.value = cfg.input || cfg.input_file;
      loadSource();
    }
  }

  if (Array.isArray(cfg.segments)) {
    segments = cfg.segments.map(s => ({
      start: String(s.start || '00:00'),
      end: String(s.end || '00:30'),
      title: s.title || '',
      subtitles: s.subtitles || null,
      sub_position: s.sub_position || 'bottom',
      sub_preset: s.sub_preset || 'hormozi',
      sub_font_size: s.sub_font_size || 48
    }));
    renderSegments();
  }

  // Render settings
  if (cfg.mode && document.getElementById('cfgMode')) document.getElementById('cfgMode').value = cfg.mode;
  if (cfg.output && document.getElementById('cfgOutputFile')) document.getElementById('cfgOutputFile').value = cfg.output;
  if (cfg.output_dir && document.getElementById('cfgOutputDir')) document.getElementById('cfgOutputDir').value = cfg.output_dir;
  if (cfg.strategy && document.getElementById('cfgStrategy')) document.getElementById('cfgStrategy').value = cfg.strategy;
  if (cfg.quality && document.getElementById('cfgQuality')) document.getElementById('cfgQuality').value = cfg.quality;
  if (typeof cfg.shorts === 'boolean' && document.getElementById('cfgShorts')) document.getElementById('cfgShorts').checked = cfg.shorts;
  if (cfg.shorts_style && document.getElementById('cfgShortsStyle')) document.getElementById('cfgShortsStyle').value = cfg.shorts_style;
  if (typeof cfg.subtitles === 'boolean' && document.getElementById('cfgBurnSubs')) document.getElementById('cfgBurnSubs').checked = cfg.subtitles;
  if (cfg.sub_preset && document.getElementById('cfgSubPreset')) document.getElementById('cfgSubPreset').value = cfg.sub_preset;
  if (cfg.sub_sdh_mode && document.getElementById('cfgSubSDHMode')) document.getElementById('cfgSubSDHMode').value = cfg.sub_sdh_mode;
  if (typeof cfg.loudnorm === 'boolean' && document.getElementById('cfgLoudnorm')) document.getElementById('cfgLoudnorm').checked = cfg.loudnorm;
  if (typeof cfg.jump_cut === 'boolean' && document.getElementById('cfgJumpCut')) document.getElementById('cfgJumpCut').checked = cfg.jump_cut;
  if (cfg.jump_cut_min_silence && document.getElementById('cfgJumpCutMinSil')) document.getElementById('cfgJumpCutMinSil').value = cfg.jump_cut_min_silence;
  if (cfg.jump_cut_margin && document.getElementById('cfgJumpCutMargin')) document.getElementById('cfgJumpCutMargin').value = cfg.jump_cut_margin;
  if (cfg.jump_cut_noise && document.getElementById('cfgJumpCutNoise')) document.getElementById('cfgJumpCutNoise').value = cfg.jump_cut_noise;
  if (cfg.watermark && document.getElementById('cfgWatermark')) document.getElementById('cfgWatermark').value = cfg.watermark;
  if (cfg.watermark_pos && document.getElementById('cfgWatermarkPos')) document.getElementById('cfgWatermarkPos').value = cfg.watermark_pos;
  if (cfg.overlay_text && document.getElementById('cfgOverlayText')) document.getElementById('cfgOverlayText').value = cfg.overlay_text;
  if (cfg.text_pos && document.getElementById('cfgTextPos')) document.getElementById('cfgTextPos').value = cfg.text_pos;
  if (cfg.font_color && document.getElementById('cfgFontColor')) document.getElementById('cfgFontColor').value = cfg.font_color;
  if (typeof cfg.generate_metadata === 'boolean' && document.getElementById('cfgMetadata')) document.getElementById('cfgMetadata').checked = cfg.generate_metadata;
  if (typeof cfg.extract_thumbnail === 'boolean' && document.getElementById('cfgThumbnail')) document.getElementById('cfgThumbnail').checked = cfg.extract_thumbnail;
  if (cfg.hwaccel && document.getElementById('cfgHwaccel')) document.getElementById('cfgHwaccel').value = cfg.hwaccel;

  if (Array.isArray(cfg.ai_configs) && cfg.ai_configs.length > 0) {
    localStorage.setItem('clipper_ai_configs', JSON.stringify(cfg.ai_configs));
  }
  if (cfg.routing_models && typeof cfg.routing_models === 'object') {
    localStorage.setItem('clipper_routing_models', JSON.stringify(cfg.routing_models));
  }
  if (cfg.ai_config?.api_key) {
    localStorage.setItem('clipper_gemini_api_key', cfg.ai_config.api_key);
  }
  initGlobalAISettings();
}

function exportCurrentConfig() {
  const currentConfig = {
    input_file: document.getElementById('videoSource')?.value.trim() || '',
    output_dir: document.getElementById('cfgOutputDir')?.value.trim() || './clips',
    output: document.getElementById('cfgOutputFile')?.value.trim() || 'merged_highlight.mp4',
    mode: document.getElementById('cfgMode')?.value || 'split',
    strategy: document.getElementById('cfgStrategy')?.value || 'fast',
    shorts: document.getElementById('cfgShorts')?.checked || false,
    shorts_style: document.getElementById('cfgShortsStyle')?.value || 'smart-crop',
    quality: document.getElementById('cfgQuality')?.value || 'best',
    subtitles: document.getElementById('cfgBurnSubs')?.checked || false,
    sub_preset: document.getElementById('cfgSubPreset')?.value || 'hormozi',
    sub_sdh_mode: document.getElementById('cfgSubSDHMode')?.value || 'strip',
    sub_emoji: true,
    loudnorm: document.getElementById('cfgLoudnorm')?.checked || false,
    jump_cut: document.getElementById('cfgJumpCut')?.checked || false,
    jump_cut_min_silence: parseFloat(document.getElementById('cfgJumpCutMinSil')?.value || '1.0'),
    jump_cut_margin: parseFloat(document.getElementById('cfgJumpCutMargin')?.value || '0.2'),
    jump_cut_noise: parseFloat(document.getElementById('cfgJumpCutNoise')?.value || '-30'),
    watermark: document.getElementById('cfgWatermark')?.value.trim() || '',
    watermark_pos: document.getElementById('cfgWatermarkPos')?.value || 'top-right',
    overlay_text: document.getElementById('cfgOverlayText')?.value.trim() || '',
    text_pos: document.getElementById('cfgTextPos')?.value || 'bottom-center',
    font_color: document.getElementById('cfgFontColor')?.value || 'white',
    generate_metadata: document.getElementById('cfgMetadata')?.checked || false,
    extract_thumbnail: document.getElementById('cfgThumbnail')?.checked || false,
    thumbnail_count: 1,
    hwaccel: document.getElementById('cfgHwaccel')?.value || 'auto',
    ai_configs: getGlobalAIConfigs(),
    routing_models: getGlobalRoutingModels(),
    segments: segments
  };

  const dataStr = "data:text/json;charset=utf-8," + encodeURIComponent(JSON.stringify(currentConfig, null, 2));
  const dlAnchor = document.createElement('a');
  dlAnchor.setAttribute("href", dataStr);
  dlAnchor.setAttribute("download", "config.json");
  document.body.appendChild(dlAnchor);
  dlAnchor.click();
  dlAnchor.remove();
}

function startPollingStatus() {
  if (pollTimer) clearInterval(pollTimer);
  pollTimer = setInterval(async () => {
    try {
      const res = await fetch('/api/status');
      const st = await res.json();

      const statusBadge = document.getElementById('statusBadge');
      const statusPill = document.getElementById('statusPill');
      const progressTitle = document.getElementById('progressTitle');
      const progressPct = document.getElementById('progressPct');
      const progressFill = document.getElementById('progressFill');

      if (st.is_running) {
        if (statusBadge) statusBadge.textContent = 'RENDERING (' + st.progress_pct + '%)';
        if (statusPill) statusPill.classList.add('rendering');
        if (progressTitle) progressTitle.textContent = st.current_task || 'Rendering...';
        if (progressPct) progressPct.textContent = st.progress_pct + '%';
        if (progressFill) progressFill.style.width = st.progress_pct + '%';
      } else {
        if (statusBadge) statusBadge.textContent = 'READY';
        if (statusPill) statusPill.classList.remove('rendering');
        if (st.last_error) {
          if (progressTitle) progressTitle.textContent = 'Failed: ' + st.last_error;
        } else if (st.completed) {
          if (progressTitle) progressTitle.textContent = 'Completed';
          if (progressFill) progressFill.style.width = '100%';
          if (progressPct) progressPct.textContent = '100%';
          switchTab('clips');
        }
        clearInterval(pollTimer);
        pollTimer = null;
      }
    } catch (err) {
      console.error('Status poll error:', err);
    }
  }, 1000);
}

function escapeHtml(str) {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#039;');
}

// Shortcuts: I = In, O = Out, Space = Play/Pause
window.addEventListener('keydown', (e) => {
  if (['INPUT', 'SELECT', 'TEXTAREA'].includes(e.target.tagName)) return;
  if (e.key === 'i' || e.key === 'I') {
    markCurrent('start');
  } else if (e.key === 'o' || e.key === 'O') {
    markCurrent('end');
  } else if (e.key === ' ') {
    e.preventDefault();
    togglePlay();
  }
});

// Subtitle Studio State & Methods
let currentEditingSegmentIdx = -1;
let currentCues = [];

function openSubtitleStudio(idx) {
  currentEditingSegmentIdx = idx;
  const s = segments[idx];
  if (!s) return;

  currentCues = (s.subtitles && Array.isArray(s.subtitles)) ? JSON.parse(JSON.stringify(s.subtitles)) : [];

  document.getElementById('subModalTitle').textContent = s.title ? s.title : `Segment #${idx + 1}`;
  document.getElementById('subModalInterval').textContent = `${s.start} → ${s.end}`;

  document.getElementById('subClipPosition').value = s.sub_position || 'bottom';
  document.getElementById('subClipPreset').value = s.sub_preset || document.getElementById('cfgSubPreset').value || 'hormozi';
  document.getElementById('subClipFontSize').value = s.sub_font_size || 48;

  setSubModalStatus(`Ready (${currentCues.length} cues loaded)`);
  renderSubtitleCues();

  const modal = document.getElementById('subModal');
  if (modal) modal.style.display = 'flex';
}

function closeSubtitleStudio() {
  const modal = document.getElementById('subModal');
  if (modal) modal.style.display = 'none';
  currentEditingSegmentIdx = -1;
  currentCues = [];
}

function setSubModalStatus(msg) {
  const el = document.getElementById('subModalStatus');
  if (el) el.textContent = msg;
}

function renderSubtitleCues() {
  const container = document.getElementById('subCuesList');
  if (!container) return;

  if (currentCues.length === 0) {
    container.innerHTML = `<div class="empty-state">No custom subtitle cues yet. Click 'Fetch / Transcribe' to load speech from YouTube/Whisper, or '+ Add Cue' to write captions manually.</div>`;
    return;
  }

  container.innerHTML = currentCues.map((c, i) => `
    <div class="sub-cue-row">
      <input type="number" step="0.1" min="0" class="text-input sub-cue-time" value="${c.start}" title="Start (sec)" onchange="updateCueTime(${i}, 'start', this.value)" />
      <span style="font-size:11px; color:var(--text-muted);">&rarr;</span>
      <input type="number" step="0.1" min="0" class="text-input sub-cue-time" value="${c.end}" title="End (sec)" onchange="updateCueTime(${i}, 'end', this.value)" />
      <input type="text" class="text-input sub-cue-text" value="${escapeHtml(c.text || '')}" placeholder="Caption line text..." oninput="updateCueText(${i}, this.value)" />
      <button class="btn btn-secondary btn-icon" style="height:28px; width:28px; padding:0;" onclick="deleteSubtitleCue(${i})" title="Delete Cue">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
  `).join('');
}

function addSubtitleCue() {
  let lastEnd = 0;
  if (currentCues.length > 0) {
    lastEnd = currentCues[currentCues.length - 1].end || 0;
  }
  currentCues.push({
    start: Math.round(lastEnd * 10) / 10,
    end: Math.round((lastEnd + 2.0) * 10) / 10,
    text: ''
  });
  renderSubtitleCues();
  setSubModalStatus(`Added cue #${currentCues.length}`);
}

function updateCueTime(idx, field, val) {
  if (!currentCues[idx]) return;
  currentCues[idx][field] = parseFloat(val) || 0;
}

function updateCueText(idx, val) {
  if (!currentCues[idx]) return;
  currentCues[idx].text = val;
}

function deleteSubtitleCue(idx) {
  currentCues.splice(idx, 1);
  renderSubtitleCues();
  setSubModalStatus(`Cue removed (${currentCues.length} remaining)`);
}

async function transcribeSegmentAudio() {
  const src = document.getElementById('videoSource').value.trim();
  if (!src) {
    alert('Please load a video source URL or local path first.');
    return;
  }
  if (currentEditingSegmentIdx < 0 || !segments[currentEditingSegmentIdx]) return;

  const s = segments[currentEditingSegmentIdx];
  const btn = document.getElementById('btnTranscribe');
  const btnText = document.getElementById('transcribeBtnText');
  const useWhisper = document.getElementById('subUseWhisper').checked;
  const lang = document.getElementById('subTranslateLang').value;

  if (btn) btn.disabled = true;
  if (btnText) btnText.textContent = 'Transcribing...';
  setSubModalStatus('Transcribing speech cues for this segment...');

  try {
    const res = await fetch('/api/transcribe', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        input_file: src,
        start: s.start,
        end: s.end,
        lang: lang,
        use_whisper: useWhisper,
        api_key: getGlobalApiKey(),
        model: getGlobalSTTModel()
      })
    });
    const data = await res.json();
    if (!res.ok) {
      alert(`Transcription error: ${data.error || 'failed'}`);
      setSubModalStatus('Transcription failed.');
    } else {
      currentCues = data.cues || [];
      renderSubtitleCues();
      setSubModalStatus(`Loaded ${currentCues.length} speech cues from video`);
    }
  } catch (err) {
    console.error('Transcribe error:', err);
    alert('Error requesting speech transcription.');
    setSubModalStatus('Network or server error.');
  } finally {
    if (btn) btn.disabled = false;
    if (btnText) btnText.textContent = 'Fetch / Transcribe';
  }
}

async function runAISubtitleAction(action) {
  if (currentCues.length === 0) {
    alert('Please add or transcribe subtitle cues first.');
    return;
  }

  setSubModalStatus(`Applying AI ${action}...`);

  const payload = {
    action: action,
    cues: currentCues,
    target_lang: document.getElementById('subTranslateLang').value,
    ai_router: 'gemini',
    api_key: getGlobalApiKey(),
    model: getGlobalSegmentModel()
  };

  try {
    const res = await fetch('/api/ai/subtitles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    const data = await res.json();
    if (!res.ok) {
      alert(`AI error: ${data.error || 'failed'}`);
      setSubModalStatus('AI processing failed.');
    } else {
      currentCues = data.cues || currentCues;
      renderSubtitleCues();
      setSubModalStatus(`AI ${action} completed on ${currentCues.length} cues`);
    }
  } catch (err) {
    console.error('AI subtitle error:', err);
    alert('Error communicating with AI service.');
    setSubModalStatus('AI error.');
  }
}

function saveSubtitleStudio() {
  if (currentEditingSegmentIdx < 0 || !segments[currentEditingSegmentIdx]) return;

  const s = segments[currentEditingSegmentIdx];
  s.subtitles = currentCues;
  s.sub_position = document.getElementById('subClipPosition').value;
  s.sub_preset = document.getElementById('subClipPreset').value;
  s.sub_font_size = parseInt(document.getElementById('subClipFontSize').value, 10) || 48;

  closeSubtitleStudio();
  renderSegments();
}

// Global Multi-AI Settings State & LocalStorage Persistence
function getGlobalAIConfigs() {
  try {
    const raw = localStorage.getItem('clipper_ai_configs');
    if (raw) {
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed) && parsed.length > 0) return parsed;
    }
  } catch (e) {
    console.error('Error reading clipper_ai_configs:', e);
  }

  // Fallback default profile if legacy key exists
  const legacyKey = localStorage.getItem('clipper_gemini_api_key') || '';
  return [
    {
      id: 'default_gemini',
      router: 'gemini',
      model: 'gemini-2.5-flash',
      key: legacyKey
    }
  ];
}

function getGlobalRoutingModels() {
  try {
    const raw = localStorage.getItem('clipper_routing_models');
    if (raw) {
      const parsed = JSON.parse(raw);
      if (parsed && typeof parsed === 'object') return parsed;
    }
  } catch (e) {
    console.error('Error reading clipper_routing_models:', e);
  }

  return {
    segment: 'default_gemini',
    sub_translate: 'default_gemini',
    metadata: 'default_gemini'
  };
}

function initGlobalAISettings() {
  renderAIProfilesList();
  populateRoutingDropdowns();
}

function renderAIProfilesList() {
  const container = document.getElementById('aiProfilesListContainer');
  if (!container) return;

  const profiles = getGlobalAIConfigs();
  if (profiles.length === 0) {
    container.innerHTML = `<div style="padding:10px; font-size:11.5px; color:var(--text-muted); text-align:center; background:var(--bg-surface); border-radius:6px;">No AI profiles configured yet. Click '+ Add Profile' to connect an AI key.</div>`;
    return;
  }

  container.innerHTML = profiles.map(p => {
    const maskedKey = p.key ? (p.key.length > 8 ? p.key.substring(0, 4) + '...' + p.key.substring(p.key.length - 4) : '••••••••') : '(no key)';
    return `
      <div style="display:flex; justify-content:space-between; align-items:center; background:var(--bg-surface); border:1px solid var(--border); padding:8px 12px; border-radius:6px;">
        <div style="display:flex; flex-direction:column; gap:2px;">
          <div style="display:flex; align-items:center; gap:6px;">
            <span style="font-size:12px; font-weight:700; color:var(--text-primary); font-family:var(--font-mono);">${escapeHtml(p.id)}</span>
            <span style="font-size:10px; padding:1px 5px; border-radius:4px; background:var(--bg-surface-elevated); border:1px solid var(--border); text-transform:uppercase; color:var(--accent-primary); font-weight:600;">${escapeHtml(p.router)}</span>
          </div>
          <span style="font-size:11px; color:var(--text-muted); font-family:var(--font-mono);">${escapeHtml(p.model || 'auto')} • ${escapeHtml(maskedKey)}</span>
        </div>
        <button class="btn btn-secondary btn-icon" style="height:26px; width:26px; padding:0; color:#ef4444;" onclick="deleteAIProfile('${escapeHtml(p.id)}')" title="Delete Profile">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px; height:14px;"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
        </button>
      </div>
    `;
  }).join('');
}

function populateRoutingDropdowns() {
  const profiles = getGlobalAIConfigs();
  const routing = getGlobalRoutingModels();

  const segSelect = document.getElementById('routeSegmentProfile');
  const subSelect = document.getElementById('routeSubtitleProfile');
  const metaSelect = document.getElementById('routeMetadataProfile');

  const optionsHTML = profiles.map(p => `
    <option value="${escapeHtml(p.id)}">${escapeHtml(p.id)} (${p.router}: ${p.model || 'default'})</option>
  `).join('');

  if (segSelect) {
    segSelect.innerHTML = optionsHTML;
    if (routing.segment && profiles.some(p => p.id === routing.segment)) {
      segSelect.value = routing.segment;
    } else if (profiles.length > 0) {
      segSelect.value = profiles[0].id;
    }
  }

  if (subSelect) {
    subSelect.innerHTML = optionsHTML;
    if (routing.sub_translate && profiles.some(p => p.id === routing.sub_translate)) {
      subSelect.value = routing.sub_translate;
    } else if (profiles.length > 0) {
      subSelect.value = profiles[0].id;
    }
  }

  if (metaSelect) {
    metaSelect.innerHTML = optionsHTML;
    if (routing.metadata && profiles.some(p => p.id === routing.metadata)) {
      metaSelect.value = routing.metadata;
    } else if (profiles.length > 0) {
      metaSelect.value = profiles[0].id;
    }
  }
}

function saveGlobalAIRouting() {
  const routing = {
    segment: document.getElementById('routeSegmentProfile')?.value || '',
    sub_translate: document.getElementById('routeSubtitleProfile')?.value || '',
    metadata: document.getElementById('routeMetadataProfile')?.value || ''
  };
  localStorage.setItem('clipper_routing_models', JSON.stringify(routing));

  const statusEl = document.getElementById('aiModalStatus');
  if (statusEl) {
    statusEl.textContent = 'Task routing updated ✓';
  }
}

function showAddProfileForm() {
  const form = document.getElementById('addProfileFormBox');
  if (form) {
    form.style.display = 'block';
    document.getElementById('newProfID').value = 'gemini_' + (getGlobalAIConfigs().length + 1);
    document.getElementById('newProfRouter').value = 'gemini';
    document.getElementById('newProfModel').value = 'gemini-2.5-flash';
    document.getElementById('newProfKey').value = '';
  }
}

function hideAddProfileForm() {
  const form = document.getElementById('addProfileFormBox');
  if (form) form.style.display = 'none';
}

function onProfileRouterChange() {
  const router = document.getElementById('newProfRouter')?.value || 'gemini';
  const modelInput = document.getElementById('newProfModel');
  if (!modelInput) return;
  if (router === 'gemini') modelInput.value = 'gemini-2.5-flash';
  else if (router === 'deepseek') modelInput.value = 'deepseek-chat';
  else if (router === 'openrouter') modelInput.value = 'openrouter/free';
  else if (router === 'openai') modelInput.value = 'gpt-4o-mini';
}

function saveNewProfile() {
  const id = document.getElementById('newProfID')?.value.trim() || '';
  const router = document.getElementById('newProfRouter')?.value || 'gemini';
  const model = document.getElementById('newProfModel')?.value.trim() || '';
  const key = document.getElementById('newProfKey')?.value.trim() || '';

  if (!id) {
    alert('Please enter a unique Profile ID.');
    return;
  }

  const profiles = getGlobalAIConfigs();
  const existingIdx = profiles.findIndex(p => p.id.toLowerCase() === id.toLowerCase());
  const newProfile = { id, router, model, key };

  if (existingIdx >= 0) {
    profiles[existingIdx] = newProfile;
  } else {
    profiles.push(newProfile);
  }

  localStorage.setItem('clipper_ai_configs', JSON.stringify(profiles));
  if (router === 'gemini' && key) {
    localStorage.setItem('clipper_gemini_api_key', key);
  }

  hideAddProfileForm();
  initGlobalAISettings();
  saveGlobalAIRouting();

  const statusEl = document.getElementById('aiModalStatus');
  if (statusEl) {
    statusEl.textContent = `Profile '${id}' saved ✓`;
  }
}

function deleteAIProfile(id) {
  let profiles = getGlobalAIConfigs();
  profiles = profiles.filter(p => p.id !== id);
  localStorage.setItem('clipper_ai_configs', JSON.stringify(profiles));
  initGlobalAISettings();
  saveGlobalAIRouting();
}

function getGlobalApiKey() {
  const profiles = getGlobalAIConfigs();
  if (profiles.length > 0 && profiles[0].key) return profiles[0].key;
  return (localStorage.getItem('clipper_gemini_api_key') || '').trim();
}

function getGlobalSegmentModel() {
  const routing = getGlobalRoutingModels();
  const profiles = getGlobalAIConfigs();
  const target = profiles.find(p => p.id === routing.segment);
  return target?.model || 'gemini-2.5-flash';
}

function getGlobalSTTModel() {
  return 'gemini-2.5-flash';
}

function toggleGlobalAISettings() {
  const modal = document.getElementById('aiModal');
  if (!modal) return;
  modal.style.display = modal.style.display === 'none' ? 'flex' : 'none';
  initGlobalAISettings();
}

function closeGlobalAISettings() {
  saveGlobalAIRouting();
  const modal = document.getElementById('aiModal');
  if (modal) modal.style.display = 'none';
}

/* ==========================================================================
   STORAGE MANAGER & CACHE CLEANUP
   ========================================================================== */
function toggleStorageModal() {
  const modal = document.getElementById('storageModal');
  if (!modal) return;
  const isOpening = modal.style.display === 'none' || modal.style.display === '';
  modal.style.display = isOpening ? 'flex' : 'none';
  if (isOpening) {
    fetchStorageStats();
  }
}

function closeStorageModal() {
  const modal = document.getElementById('storageModal');
  if (modal) modal.style.display = 'none';
}

async function fetchStorageStats() {
  const statusEl = document.getElementById('storageModalStatus');
  if (statusEl) statusEl.textContent = 'Calculating disk metrics...';
  try {
    const res = await fetch('/api/storage/stats');
    const data = await res.json();
    if (res.ok) {
      const cacheDirEl = document.getElementById('storageCacheDir');
      const cacheSizeEl = document.getElementById('storageCacheSize');
      const cacheCountEl = document.getElementById('storageCacheCount');
      if (cacheDirEl) cacheDirEl.textContent = `Directory: ${data.cache_dir}`;
      if (cacheSizeEl) cacheSizeEl.textContent = data.cache_size_str;
      if (cacheCountEl) cacheCountEl.textContent = `${data.cache_file_count} cached video/audio files`;

      const clipsDirEl = document.getElementById('storageClipsDir');
      const clipsSizeEl = document.getElementById('storageClipsSize');
      const clipsCountEl = document.getElementById('storageClipsCount');
      if (clipsDirEl) clipsDirEl.textContent = `Directory: ${data.clips_dir}`;
      if (clipsSizeEl) clipsSizeEl.textContent = data.clips_size_str;
      if (clipsCountEl) clipsCountEl.textContent = `${data.clips_count} rendered clips found`;

      if (statusEl) statusEl.textContent = 'Storage data up to date ✓';
    } else {
      if (statusEl) statusEl.textContent = 'Failed to fetch storage stats';
    }
  } catch (err) {
    if (statusEl) statusEl.textContent = 'Error: ' + err.message;
  }
}

async function purgeCache() {
  if (!confirm('Are you sure you want to purge all downloaded video and audio cache files?')) return;
  const btn = document.getElementById('purgeCacheBtn');
  if (btn) {
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner" style="width:14px;height:14px;border-width:2px;display:inline-block;vertical-align:middle;margin-right:4px;"></span> Purging...';
  }
  try {
    const res = await fetch('/api/storage/clean-cache', { method: 'POST' });
    let data = {};
    const text = await res.text();
    try {
      data = JSON.parse(text);
    } catch {
      data = { error: text || `HTTP ${res.status} ${res.statusText}` };
    }

    if (!res.ok) {
      alert(data.error || 'Failed to purge cache.');
      return;
    }
    fetchStorageStats();
    alert(data.message || 'Cache purged successfully.');
  } catch (err) {
    console.error('Purge cache error:', err);
    alert('Failed to connect to Clipper server. Please make sure `./clipper serve :8000` is running in your terminal.');
  } finally {
    if (btn) {
      btn.disabled = false;
      btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg><span>Purge Cache</span>';
    }
  }
}

async function cleanAllClips() {
  if (!confirm('⚠️ Are you sure you want to DELETE ALL rendered clips and thumbnails? This action cannot be undone.')) return;
  const btn = document.getElementById('cleanClipsBtn');
  const topBtn = document.getElementById('clearAllClipsBtn');
  if (btn) btn.disabled = true;
  if (topBtn) topBtn.disabled = true;

  try {
    const res = await fetch('/api/storage/clean-clips', { method: 'POST' });
    let data = {};
    const text = await res.text();
    try {
      data = JSON.parse(text);
    } catch {
      data = { error: text || `HTTP ${res.status} ${res.statusText}` };
    }

    if (!res.ok) {
      alert(data.error || 'Failed to clean clips.');
      return;
    }
    loadClipsGallery();
    fetchStorageStats();
    alert(data.message || 'All clips deleted successfully.');
  } catch (err) {
    console.error('Clean clips error:', err);
    alert('Failed to connect to Clipper server. Please make sure `./clipper serve :8000` is running in your terminal.');
  } finally {
    if (btn) btn.disabled = false;
    if (topBtn) topBtn.disabled = false;
  }
}

// Init on load
document.addEventListener('DOMContentLoaded', () => {
  initTheme();
  initGlobalAISettings();
  setupPlayerEvents();
  loadClipsGallery();
});

// Run immediate fallback if DOM already ready
initTheme();
initGlobalAISettings();
setupPlayerEvents();
loadClipsGallery();
