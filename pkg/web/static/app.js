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

// Player time update handler
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

    player.addEventListener('play', () => {
      const ph = document.getElementById('playerPlaceholder');
      if (ph) ph.style.display = 'none';
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

async function loadSourceVideo() {
  const src = document.getElementById('videoSource').value.trim();
  if (!src) {
    alert('Please enter a local video file path or YouTube URL.');
    return;
  }

  const loader = document.getElementById('playerLoader');
  const title = document.getElementById('loaderTitle');
  const sub = document.getElementById('loaderSub');
  const loadBtn = document.getElementById('loadBtn');
  const loadBtnText = document.getElementById('loadBtnText');
  const ph = document.getElementById('playerPlaceholder');

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
    if (sub) sub.textContent = 'Loading local video file into browser preview player.';
  }

  try {
    const res = await fetch('/api/prepare', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source: src })
    });
    const data = await res.json();

    if (!res.ok) {
      alert(data.error || 'Failed to prepare video.');
      if (loader) loader.style.display = 'none';
      if (loadBtn) loadBtn.disabled = false;
      if (loadBtnText) loadBtnText.textContent = 'Load';
      return;
    }

    if (!player) player = document.getElementById('videoPlayer');
    if (player) {
      player.src = data.preview_url;
      player.load();
      if (ph) ph.style.display = 'none';
      player.oncanplay = () => {
        if (loader) loader.style.display = 'none';
        player.oncanplay = null;
      };
      // Fallback timeout to hide loader
      setTimeout(() => {
        if (loader) loader.style.display = 'none';
      }, 4000);
    } else {
      if (loader) loader.style.display = 'none';
    }
  } catch (err) {
    alert('Error preparing video: ' + err);
    if (loader) loader.style.display = 'none';
  } finally {
    if (loadBtn) loadBtn.disabled = false;
    if (loadBtnText) loadBtnText.textContent = 'Load';
  }
}

async function loadClipsGallery() {
  try {
    const res = await fetch('/api/clips');
    const data = await res.json();
    const grid = document.getElementById('clipsGrid');
    const countEl = document.getElementById('clipsCount');
    if (countEl) countEl.textContent = data ? data.length : 0;

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
          ? `<img src="${c.thumbnail_url}" class="clip-thumb" alt="${escapeHtml(c.name)}" />`
          : `<div class="clip-thumb-empty"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/><line x1="7" y1="2" x2="7" y2="22"/><line x1="17" y1="2" x2="17" y2="22"/><line x1="2" y1="12" x2="22" y2="12"/></svg></div>`
        }
        <div class="clip-details">
          <div class="clip-name" title="${escapeHtml(c.name)}">${escapeHtml(c.name)}</div>
          <div class="clip-meta-row">
            <span>${c.size_str}</span>
            <span>${c.mod_time || ''}</span>
          </div>
          <div class="clip-actions">
            <a href="${c.url}" target="_blank" class="btn btn-secondary btn-sm" style="flex:1; text-decoration:none;">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
              Play
            </a>
            <a href="${c.url}" download class="btn btn-secondary btn-sm" title="Download">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            </a>
          </div>
        </div>
      </div>
    `).join('');
  } catch (err) {
    console.error('Error loading clips:', err);
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
    const data = await res.json();
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

  if (cfg.ai_config?.api_key) {
    localStorage.setItem('clipper_gemini_api_key', cfg.ai_config.api_key);
    initGlobalAISettings();
  }
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

// Global AI Settings State & LocalStorage Persistence
function initGlobalAISettings() {
  const savedKey = localStorage.getItem('clipper_gemini_api_key') || '';
  const savedSegModel = localStorage.getItem('clipper_ai_segment_model') || 'gemini-3.8-flash';
  const savedSTTModel = localStorage.getItem('clipper_ai_stt_model') || 'gemini-3.5-transcribe';

  const keyEl = document.getElementById('globalAIApiKey');
  const segEl = document.getElementById('globalAISegmentModel');
  const sttEl = document.getElementById('globalAISTTModel');

  if (keyEl) keyEl.value = savedKey;
  if (segEl) segEl.value = savedSegModel;
  if (sttEl) sttEl.value = savedSTTModel;
}

function getGlobalApiKey() {
  return (document.getElementById('globalAIApiKey')?.value || localStorage.getItem('clipper_gemini_api_key') || '').trim();
}

function getGlobalSegmentModel() {
  return document.getElementById('globalAISegmentModel')?.value || localStorage.getItem('clipper_ai_segment_model') || 'gemini-3.8-flash';
}

function getGlobalSTTModel() {
  return document.getElementById('globalAISTTModel')?.value || localStorage.getItem('clipper_ai_stt_model') || 'gemini-3.5-transcribe';
}

function saveGlobalAISettings() {
  const key = document.getElementById('globalAIApiKey')?.value.trim() || '';
  const segModel = document.getElementById('globalAISegmentModel')?.value || 'gemini-3.8-flash';
  const sttModel = document.getElementById('globalAISTTModel')?.value || 'gemini-3.5-transcribe';

  localStorage.setItem('clipper_gemini_api_key', key);
  localStorage.setItem('clipper_ai_segment_model', segModel);
  localStorage.setItem('clipper_ai_stt_model', sttModel);

  const statusEl = document.getElementById('aiModalStatus');
  if (statusEl) {
    statusEl.textContent = key ? 'API key saved locally ✓' : 'Ready (No API key set)';
  }
}

function toggleGlobalAISettings() {
  const modal = document.getElementById('aiModal');
  if (!modal) return;
  modal.style.display = modal.style.display === 'none' ? 'flex' : 'none';
  initGlobalAISettings();
}

function closeGlobalAISettings() {
  saveGlobalAISettings();
  const modal = document.getElementById('aiModal');
  if (modal) modal.style.display = 'none';
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
