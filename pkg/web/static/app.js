// Clipper Studio - Client Application
const player = document.getElementById('mainPlayer');
let segments = [];
let pollTimer = null;

// Player event listeners
if (player) {
  player.addEventListener('timeupdate', () => {
    const cur = player.currentTime || 0;
    const dur = player.duration || 0;
    const el = document.getElementById('playheadTime');
    if (el) {
      el.textContent = formatTimecode(cur) + ' / ' + formatTimecode(dur);
    }
  });
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
  if (player && !isNaN(player.duration)) {
    player.currentTime = Math.max(0, Math.min(player.duration, player.currentTime + delta));
  }
}

function togglePlay() {
  if (!player) return;
  if (player.paused) {
    player.play();
  } else {
    player.pause();
  }
}

function markCurrent(target) {
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
    alert('Please enter both start and end timestamps.');
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
  if (!player) return;
  player.currentTime = parseSeconds(startStr);
  player.play();
}

function renderSegments() {
  const list = document.getElementById('segmentStack');
  const countEl = document.getElementById('queueCount');
  if (countEl) countEl.textContent = segments.length;

  if (!list) return;

  if (segments.length === 0) {
    list.innerHTML = '<div class="empty-state">No segments added yet. Use the trimmer below the player to define in/out points.</div>';
    return;
  }

  list.innerHTML = segments.map((s, i) => `
    <div class="segment-item">
      <div>
        <div class="segment-title">${s.title ? escapeHtml(s.title) : 'Segment #' + (i + 1)}</div>
        <div class="segment-range">${s.start} &rarr; ${s.end}</div>
      </div>
      <div style="display:flex; gap:6px;">
        <button class="btn btn-secondary btn-sm" onclick="seekToSegment('${s.start}')" title="Play Segment">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
        </button>
        <button class="btn btn-danger btn-sm" onclick="removeSegment(${i})" title="Remove">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>
    </div>
  `).join('');
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

function loadSourceVideo() {
  const src = document.getElementById('videoSource').value.trim();
  if (!src) {
    alert('Please provide a local video file path or YouTube URL.');
    return;
  }

  if (src.startsWith('http://') || src.startsWith('https://')) {
    alert('YouTube URL detected. Set timestamps manually or via player once downloaded.');
  } else {
    player.src = '/preview?path=' + encodeURIComponent(src);
    player.load();
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
      grid.innerHTML = '<div class="empty-state">No rendered clips found in output directory.</div>';
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
    console.error('Error fetching clips:', err);
  }
}

async function startClippingJob() {
  const input = document.getElementById('videoSource').value.trim();
  if (!input) {
    alert('Please specify an input video file or YouTube URL.');
    return;
  }
  if (segments.length === 0) {
    alert('Please add at least one segment to the queue before rendering.');
    return;
  }

  const payload = {
    input_file: input,
    segments: segments,
    shorts: document.getElementById('cfgShorts').checked,
    shorts_style: document.getElementById('cfgShortsStyle').value,
    burn_subtitles: document.getElementById('cfgBurnSubs').checked,
    sub_preset: document.getElementById('cfgSubPreset').value,
    sub_emoji: true,
    loudnorm: document.getElementById('cfgLoudnorm').checked,
    jump_cut: document.getElementById('cfgJumpCut').checked,
    generate_metadata: document.getElementById('cfgMetadata').checked,
    extract_thumbnail: document.getElementById('cfgThumbnail').checked,
    thumbnail_count: 1,
    hwaccel: document.getElementById('cfgHwaccel').value
  };

  try {
    const res = await fetch('/api/clip', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    const data = await res.json();
    if (!res.ok) {
      alert(data.error || 'Failed to start clipping job.');
      return;
    }

    const progressCard = document.getElementById('progressCard');
    if (progressCard) progressCard.style.display = 'block';
    startPollingStatus();
  } catch (err) {
    alert('Network error communicating with server: ' + err);
  }
}

function startPollingStatus() {
  if (pollTimer) clearInterval(pollTimer);
  pollTimer = setInterval(async () => {
    try {
      const res = await fetch('/api/status');
      const st = await res.json();

      const statusDot = document.getElementById('statusDot');
      const statusText = document.getElementById('statusText');
      const progressTitle = document.getElementById('progressTitle');
      const progressPct = document.getElementById('progressPct');
      const progressFill = document.getElementById('progressFill');

      if (st.is_running) {
        if (statusDot) statusDot.className = 'status-dot busy';
        if (statusText) statusText.textContent = 'Rendering (' + st.progress_pct + '%)';
        if (progressTitle) progressTitle.textContent = st.current_task || 'Rendering...';
        if (progressPct) progressPct.textContent = st.progress_pct + '%';
        if (progressFill) progressFill.style.width = st.progress_pct + '%';
      } else {
        if (statusDot) statusDot.className = 'status-dot';
        if (statusText) statusText.textContent = 'Ready';
        if (st.last_error) {
          if (progressTitle) progressTitle.textContent = 'Failed: ' + st.last_error;
        } else if (st.completed) {
          if (progressTitle) progressTitle.textContent = 'Rendering completed';
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

// Keyboard shortcuts: I = Mark Start, O = Mark End
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

// Initial clips fetch
loadClipsGallery();
