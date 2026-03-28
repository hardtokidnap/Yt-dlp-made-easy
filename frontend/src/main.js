import './style.css';
import {EventsOn} from '../wailsjs/runtime/runtime';
import * as App from '../wailsjs/go/main/App';

// State management
const state = {
    currentTab: 'download',
    downloads: {},
    history: [],
    recentHistory: [],
    settings: null,
    logs: [],
    conversion: null,
    ffmpegInstalled: false
};

// Tab definitions
const tabs = [
    {id: 'download', label: 'Download', icon: '⬇️'},
    {id: 'convert', label: 'Convert', icon: '🔄'},
    {id: 'history', label: 'History', icon: '📜'},
    {id: 'settings', label: 'Settings', icon: '⚙️'},
    {id: 'log', label: 'Log', icon: '📋'}
];

// Initialize app
async function init() {
    renderTabs();
    setupEventListeners(); // Set up listeners first
    await loadSettings();
    await loadQueueStatus(); // Load any existing downloads
    await loadRecentHistory();
    switchTab('download');
}


// Load current queue status
async function loadQueueStatus() {
    try {
        const items = await App.GetQueueStatus();
        if (items && items.length > 0) {
            items.forEach(item => {
                state.downloads[item.id] = item;
            });
            if (state.currentTab === 'download') {
                updateDownloadQueue();
            }
        }
    } catch (err) {
        console.error('Failed to load queue status:', err);
    }
}

async function loadRecentHistory() {
    try {
        state.recentHistory = await App.GetRecentHistory(3) || [];
    } catch (_) {
        state.recentHistory = [];
    }
}

// Render tab buttons
function renderTabs() {
    const tabsContainer = document.getElementById('tabs');
    tabsContainer.innerHTML = tabs.map(tab => `
        <button
            id="tab-${tab.id}"
            onclick="window.switchTab('${tab.id}')"
            class="tab-button ${tab.id === state.currentTab ? 'tab-button-active' : 'tab-button-inactive'}"
        >
            ${tab.icon} ${tab.label}
        </button>
    `).join('');
}

// Switch active tab
window.switchTab = function(tabId) {
    state.currentTab = tabId;
    renderTabs();

    const contentMap = {
        'download': renderDownloadTab,
        'convert': renderConvertTab,
        'history': renderHistoryTab,
        'settings': renderSettingsTab,
        'log': renderLogTab
    };

    contentMap[tabId]();
};

// Download Tab
function renderDownloadTab() {
    const content = document.getElementById('tab-content');
    content.innerHTML = `
        <div class="max-w-4xl mx-auto space-y-6">
            <!-- URL Input -->
            <div class="card">
                <h2 class="text-xl font-semibold mb-4">Add Download</h2>
                <div class="space-y-4">
                    <div>
                        <label class="block text-sm font-medium mb-2">Video URL</label>
                        <textarea
                            id="url-input"
                            rows="3"
                            class="input-field resize-none"
                            placeholder="https://www.youtube.com/watch?v=..."
                        ></textarea>
                    </div>

                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label class="block text-sm font-medium mb-2">Quality</label>
                            <select id="quality-select" class="select-field w-full">
                                <option value="best">Best</option>
                                <option value="4K">4K (2160p)</option>
                                <option value="1440p">1440p</option>
                                <option value="1080p">1080p</option>
                                <option value="720p">720p</option>
                                <option value="480p">480p</option>
                                <option value="360p">360p</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium mb-2">Format</label>
                            <select id="format-select" class="select-field w-full">
                                <option value="best">Best</option>
                                <option value="mp4">MP4</option>
                                <option value="mkv">MKV</option>
                                <option value="webm">WebM</option>
                            </select>
                        </div>
                    </div>

                    <div class="flex items-center space-x-4">
                        <label class="flex items-center space-x-2 cursor-pointer">
                            <input type="checkbox" id="audio-only" class="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600 focus:ring-blue-500 focus:ring-2">
                            <span class="text-sm">Audio Only</span>
                        </label>
                    </div>

                    <div class="flex space-x-3">
                        <button onclick="window.startDownload()" class="btn-primary flex-1">
                            Start Download
                        </button>
                        <button onclick="window.selectFolder()" class="btn-secondary">
                            📁 Choose Folder
                        </button>
                    </div>

                    <div id="save-path-display" class="text-sm text-gray-400"></div>
                </div>
            </div>

            <!-- Download Queue -->
            <div class="card">
                <div class="flex justify-between items-center mb-4">
                    <h2 class="text-xl font-semibold">Download Queue</h2>
                    <button onclick="window.clearQueue()" class="btn-secondary text-sm">
                        🗑️ Clear Queue
                    </button>
                </div>
                <div id="download-queue" class="space-y-3">
                    <p class="text-gray-400 text-center py-8">No active downloads</p>
                </div>
            </div>
        </div>
    `;

    updateDownloadQueue();
    displaySavePath();
}

// Start download
window.startDownload = async function() {
    const url = document.getElementById('url-input').value.trim();
    if (!url) {
        alert('Please enter a URL');
        return;
    }

    const audioOnly = document.getElementById('audio-only').checked;
    const quality = document.getElementById('quality-select').value;
    const format = document.getElementById('format-select').value;

    const opts = {
        is_audio_only: audioOnly,
        quality: quality,
        format: format
    };

    try {
        const id = await App.AddDownload(url, opts);
        document.getElementById('url-input').value = '';
        addLog(`✅ Added download: ${url}`);

        // Switch to download tab to see progress
        if (state.currentTab !== 'download') {
            switchTab('download');
        }
    } catch (err) {
        alert('Failed to add download: ' + err);
        addLog(`❌ Error adding download: ${err}`);
    }
};

// Select save folder
window.selectFolder = async function() {
    try {
        const path = await App.BrowseFolder();
        if (path) {
            // Reload settings to get updated path
            await loadSettings();
            displaySavePath();

            // Update settings display if on settings tab
            if (state.currentTab === 'settings') {
                const folderInput = document.getElementById('save-folder');
                if (folderInput) {
                    folderInput.value = path;
                }
            }
        }
    } catch (err) {
        console.error('Failed to select folder:', err);
    }
};

// Display current save path
async function displaySavePath() {
    const display = document.getElementById('save-path-display');
    if (display) {
        try {
            const settings = await App.GetSettings();
            display.textContent = `Save to: ${settings.General.SaveFolder}`;
        } catch (err) {
            display.textContent = '';
        }
    }
}

// Update download queue display — active items from queue, recent from history
function updateDownloadQueue() {
    const queue = document.getElementById('download-queue');
    if (!queue) return;

    const active = Object.values(state.downloads);

    let html = active.map(item => {
        if (item.status === 'error') return renderErrorCard(item);
        return renderDownloadCard(item);
    }).join('');

    const recent = state.recentHistory || [];
    if (recent.length > 0) {
        html += recent.map(entry => renderHistoryCard(entry)).join('');
    }

    if (!html) {
        queue.innerHTML = '<p class="text-gray-400 text-center py-8">No downloads yet</p>';
        return;
    }
    queue.innerHTML = html;
}

function renderHistoryCard(entry) {
    const badge = entry.is_audio_only
        ? '<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-900/50 text-purple-300 border border-purple-700/50">🎵 Audio</span>'
        : '<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-900/50 text-blue-300 border border-blue-700/50">🎬 Video</span>';

    const isError = entry.status === 'error';
    const fileMissing = !isError && entry.file_path && entry.file_exists === false;
    const title = entry.title || '';

    return `
        <div class="bg-gray-700 rounded-lg p-4 space-y-2${isError ? ' border border-red-800/50' : ''}${fileMissing ? ' opacity-50' : ''}">
            <div class="flex justify-between items-start">
                <div class="flex-1 min-w-0 mr-4">
                    <div class="flex items-center gap-2 min-w-0">
                        ${badge}
                        <span class="font-medium truncate">${escapeHtml(title || entry.url)}</span>
                    </div>
                    ${title && entry.url ? `
                        <div class="text-xs text-gray-500 truncate cursor-pointer hover:text-blue-400 mt-1"
                             onclick="window.openURL('${escapeJsStr(entry.url)}')">${escapeHtml(entry.url)}</div>
                    ` : ''}
                </div>
                <div class="flex items-center space-x-2 shrink-0">
                    ${!isError && entry.file_path && !fileMissing ? `
                        <button onclick="window.openFileInFolder('${escapeJsStr(entry.file_path)}')" class="btn-secondary text-xs">
                            📂 Open Folder
                        </button>
                    ` : ''}
                    ${fileMissing ? `
                        <span class="text-xs text-gray-500 italic">File missing</span>
                    ` : ''}
                    ${entry.url ? `
                        <button onclick="window.redownload('${escapeJsStr(entry.url)}')" class="btn-secondary text-xs">
                            🔄 Re-download
                        </button>
                    ` : ''}
                    <button onclick="window.hideFromQueue('${entry.id}')" class="text-gray-500 hover:text-gray-300 text-lg leading-none" title="Dismiss">✕</button>
                </div>
            </div>
            <div class="flex justify-between text-sm text-gray-300">
                <span>${isError ? `<span class="text-red-400">Error: ${escapeHtml(entry.error || 'Unknown')}</span>` : '<span class="text-green-400">Completed</span>'}</span>
                <span class="text-gray-500 text-xs">${entry.date ? new Date(entry.date).toLocaleString() : ''}</span>
            </div>
        </div>
    `;
}

// Render a normal download card
function renderDownloadCard(item) {
    const badge = item.is_audio_only
        ? '<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-900/50 text-purple-300 border border-purple-700/50">🎵 Audio</span>'
        : '<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-900/50 text-blue-300 border border-blue-700/50">🎬 Video</span>';

    const fileMissing = item.status === 'completed' && item.file_path && item.file_exists === false;
    const isDone = item.status === 'completed' || item.status === 'stopped';
    const title = item.title || '';

    return `
        <div class="bg-gray-700 rounded-lg p-4 space-y-2${fileMissing ? ' opacity-50' : ''}">
            <div class="flex justify-between items-start">
                <div class="flex-1 min-w-0 mr-4">
                    <div class="flex items-center gap-2 min-w-0">
                        ${badge}
                        <span class="font-medium truncate">${escapeHtml(title || item.url)}</span>
                    </div>
                    ${title && item.url ? `
                        <div class="text-xs text-gray-500 truncate cursor-pointer hover:text-blue-400 mt-1"
                             onclick="window.openURL('${escapeJsStr(item.url)}')">${escapeHtml(item.url)}</div>
                    ` : ''}
                </div>
                <div class="flex items-center space-x-2 shrink-0">
                    ${item.status === 'downloading' ? `
                        <button onclick="window.pauseDownload('${item.id}')" class="btn-secondary text-xs">
                            ⏸️ Pause
                        </button>
                    ` : ''}
                    ${item.status === 'paused' ? `
                        <button onclick="window.resumeDownload('${item.id}')" class="btn-secondary text-xs">
                            ▶️ Resume
                        </button>
                    ` : ''}
                    ${item.status === 'downloading' || item.status === 'paused' ? `
                        <button onclick="window.stopDownload('${item.id}')" class="btn-danger text-xs">
                            ⏹️ Stop
                        </button>
                    ` : ''}
                    ${item.status === 'completed' && item.file_path && !fileMissing ? `
                        <button onclick="window.openFileInFolder('${escapeJsStr(item.file_path)}')" class="btn-secondary text-xs">
                            📂 Open Folder
                        </button>
                    ` : ''}
                    ${fileMissing ? `
                        <span class="text-xs text-gray-500 italic">File missing</span>
                    ` : ''}
                    ${isDone && item.url ? `
                        <button onclick="window.redownload('${escapeJsStr(item.url)}')" class="btn-secondary text-xs">
                            🔄 Re-download
                        </button>
                    ` : ''}
                    ${isDone ? `
                        <button onclick="window.removeFromQueue('${item.id}')" class="text-gray-500 hover:text-gray-300 text-lg leading-none" title="Remove from queue">✕</button>
                    ` : ''}
                </div>
            </div>

            <div class="progress-bar">
                <div class="progress-bar-fill" style="width: ${item.progress}%"></div>
            </div>

            <div class="flex justify-between text-sm text-gray-300">
                <span>${getStatusText(item)}</span>
                <span>${item.speed || ''} ${item.eta ? `• ETA ${item.eta}` : ''}</span>
            </div>
        </div>
    `;
}

// Render an error card with suggestions
function renderErrorCard(item) {
    const errorTypeLabels = {
        'forbidden_403': '403 Forbidden',
        'rate_limit_429': 'Rate Limited',
        'age_restricted': 'Age-Restricted',
        'geo_blocked': 'Geo-Blocked',
        'not_available': 'Video Unavailable',
        'network': 'Network Error',
        'extractor_outdated': 'Extractor Issue',
        'sign_in_required': 'Sign-In Required',
        'unknown': 'Error'
    };

    // Ensure we have a valid ID
    const itemId = item.id || '';
    if (!itemId) {
        console.error('Error card rendered without valid item.id:', item);
    }

    const errorLabel = errorTypeLabels[item.error_type] || 'Error';
    const suggestions = item.suggestions || [];
    const topSuggestions = suggestions.slice(0, 2);
    const moreSuggestions = suggestions.slice(2);

    return `
        <div class="bg-red-900/30 border border-red-700/50 rounded-lg p-4 space-y-3" data-item-id="${itemId}">
            <div class="flex justify-between items-start">
                <div class="flex-1 mr-4">
                    <div class="flex items-center gap-2">
                        <span class="text-red-400">❌</span>
                        <span class="font-medium text-red-300">${errorLabel}</span>
                    </div>
                    <div class="text-sm text-gray-300 truncate mt-1">${escapeHtml(item.title || item.url)}</div>
                </div>
                <button onclick="window.dismissError('${itemId}')" class="text-gray-500 hover:text-gray-300 text-sm">
                    ✕
                </button>
            </div>

            <div class="text-sm text-gray-400 bg-gray-800/50 rounded p-2 font-mono break-all">
                ${escapeHtml(item.error || 'Unknown error')}
            </div>

            ${topSuggestions.length > 0 ? `
                <div class="space-y-2">
                    <div class="text-sm text-gray-400 flex items-center gap-1">
                        <span>💡</span> Suggested fixes:
                    </div>
                    <div class="flex flex-wrap gap-2">
                        ${topSuggestions.map((s, i) => `
                            <button
                                onclick="window.applySuggestion('${itemId}', '${s.id}', '${escapeJsStr(s.action)}', '${escapeJsStr(s.action_data || '')}')"
                                class="${i === 0 ? 'bg-blue-600 hover:bg-blue-700' : 'bg-gray-600 hover:bg-gray-500'} text-white text-sm px-3 py-1.5 rounded transition-colors"
                                title="${escapeHtml(s.description)}"
                            >
                                ${escapeHtml(s.title)}
                            </button>
                        `).join('')}
                        <button
                            onclick="window.retryDownload('${itemId}')"
                            class="bg-gray-600 hover:bg-gray-500 text-white text-sm px-3 py-1.5 rounded transition-colors"
                        >
                            🔄 Retry
                        </button>
                    </div>
                    ${moreSuggestions.length > 0 ? `
                        <details class="text-sm">
                            <summary class="text-gray-500 cursor-pointer hover:text-gray-400">
                                More options (${moreSuggestions.length})
                            </summary>
                            <div class="flex flex-wrap gap-2 mt-2">
                                ${moreSuggestions.map(s => `
                                    <button
                                        onclick="window.applySuggestion('${itemId}', '${s.id}', '${escapeJsStr(s.action)}', '${escapeJsStr(s.action_data || '')}')"
                                        class="bg-gray-700 hover:bg-gray-600 text-gray-300 text-sm px-3 py-1.5 rounded transition-colors"
                                        title="${escapeHtml(s.description)}"
                                    >
                                        ${escapeHtml(s.title)}
                                    </button>
                                `).join('')}
                            </div>
                        </details>
                    ` : ''}
                </div>
            ` : `
                <div class="flex gap-2">
                    <button
                        onclick="window.retryDownload('${itemId}')"
                        class="bg-blue-600 hover:bg-blue-700 text-white text-sm px-3 py-1.5 rounded transition-colors"
                    >
                        🔄 Retry Download
                    </button>
                </div>
            `}
        </div>
    `;
}

// Download control functions
window.pauseDownload = async function(id) {
    try {
        await App.PauseDownload(id);
    } catch (err) {
        alert('Failed to pause: ' + err);
    }
};

window.resumeDownload = async function(id) {
    try {
        await App.ResumeDownload(id);
    } catch (err) {
        alert('Failed to resume: ' + err);
    }
};

window.stopDownload = async function(id) {
    try {
        await App.StopDownload(id);
    } catch (err) {
        alert('Failed to stop: ' + err);
    }
};

// Remove an item from the queue
window.removeFromQueue = async function(id) {
    addVerboseLog(`removeFromQueue: id=${id}`);
    try {
        await App.RemoveDownload(id);
        delete state.downloads[id];
        updateDownloadQueue();
        addVerboseLog(`Removed download ${id} from queue`);
    } catch (err) {
        addLog(`⚠️ Failed to remove: ${err}`);
    }
};
window.dismissError = window.removeFromQueue;

// Clear all completed/stopped/error items from the queue
window.clearQueue = async function() {
    try {
        // Hide all visible history items from queue display
        await Promise.all(state.recentHistory.map(entry => App.HideFromQueue(entry.id)));
        // Also clear any stopped items from the active queue
        await App.ClearCompletedDownloads();
        state.downloads = {};
        await loadRecentHistory();
        updateDownloadQueue();
        addLog('Queue cleared');
    } catch (err) {
        addLog(`⚠️ Failed to clear queue: ${err}`);
    }
};

// Retry a failed download
window.retryDownload = async function(id) {
    addVerboseLog(`retryDownload called with id: ${id}`);
    addVerboseLog(`state.downloads keys: ${Object.keys(state.downloads).join(', ') || '(empty)'}`);
    const item = state.downloads[id];
    if (!item) {
        addLog(`⚠️ Cannot retry: download ${id} not found in queue`);
        return;
    }

    try {
        // Remove the failed download
        await App.RemoveDownload(id);
        delete state.downloads[id];

        // Re-add the download
        const opts = {
            is_audio_only: item.is_audio_only || false,
            quality: item.quality || 'best',
            format: item.format || 'best'
        };
        await App.AddDownload(item.url, opts);
        addLog(`🔄 Retrying download: ${item.url}`);
        updateDownloadQueue();
    } catch (err) {
        addLog(`❌ Failed to retry: ${err}`);
        alert('Failed to retry: ' + err);
    }
};

// Apply a suggestion to fix an error
window.applySuggestion = async function(itemId, suggestionId, action, actionData) {
    addVerboseLog(`applySuggestion: ${suggestionId} (${action})`);
    addVerboseLog(`itemId: ${itemId}, state keys: ${Object.keys(state.downloads).join(', ') || '(empty)'}`);
    const item = state.downloads[itemId];
    if (!item) {
        addLog(`⚠️ Cannot apply fix: download ${itemId} not found`);
        return;
    }

    addVerboseLog(`Applying fix: ${suggestionId}`);

    try {
        switch (action) {
            case 'apply_setting':
                await applySettingFix(actionData);
                addVerboseLog(`Settings applied, now retrying with id: ${itemId}`);
                // Auto-retry after applying setting
                await window.retryDownload(itemId);
                addVerboseLog(`Retry initiated`);
                break;

            case 'open_settings':
                // Switch to settings tab and scroll to section
                switchTab('settings');
                setTimeout(() => {
                    const sectionId = actionData === 'auth' ? 'cookies-browser' :
                                     actionData === 'network' ? 'rate-limit' : null;
                    if (sectionId) {
                        document.getElementById(sectionId)?.scrollIntoView({ behavior: 'smooth', block: 'center' });
                    }
                }, 100);
                break;

            case 'open_link':
                App.OpenURL(actionData);
                break;

            case 'open_log':
                switchTab('log');
                break;

            case 'retry':
                await window.retryDownload(itemId);
                break;

            case 'update_ytdlp':
                addLog('🔧 Checking for yt-dlp updates...');
                await App.UpdateYtDlp();
                addLog('✅ yt-dlp updated');
                await window.retryDownload(itemId);
                break;

            default:
                console.warn('Unknown action:', action);
        }
    } catch (err) {
        addLog(`❌ Failed to apply fix: ${err}`);
        alert('Failed to apply fix: ' + err);
    }
};

// Apply a setting fix and save
async function applySettingFix(actionData) {
    if (!state.settings) {
        await loadSettings();
    }
    if (!actionData) return;

    const [key, value] = actionData.split(':');

    switch (key) {
        case 'player_client':
            state.settings.Auth.PlayerClient = value;
            addVerboseLog(`Changed player client to: ${value}`);
            break;

        case 'use_nightly':
            state.settings.Advanced.UseNightly = value === 'true';
            addVerboseLog(`${value === 'true' ? 'Enabled' : 'Disabled'} nightly builds`);
            break;

        case 'max_concurrent':
            state.settings.General.MaxConcurrentDownloads = parseInt(value) || 1;
            addVerboseLog(`Set max concurrent downloads to: ${value}`);
            break;

        default:
            console.warn('Unknown setting key:', key);
            return;
    }

    // Save the settings
    await App.SaveSettings(state.settings);
    addVerboseLog('Settings saved via applySettingFix');
}

// Get status display text
function getStatusText(item) {
    const statusMap = {
        'pending': 'Pending',
        'downloading': `Downloading ${(item.progress || 0).toFixed(1)}%`,
        'paused': 'Paused',
        'stopped': 'Stopped',
        'completed': 'Completed',
        'error': `Error: ${item.error || 'Unknown'}`
    };
    return statusMap[item.status] || item.status;
}

// Convert Tab (FFmpeg)
async function renderConvertTab() {
    const content = document.getElementById('tab-content');

    let ffmpegInstalled = false;
    try {
        ffmpegInstalled = await App.IsFFmpegInstalled();
    } catch (_) { /* ignore */ }

    if (!ffmpegInstalled) {
        content.innerHTML = `
            <div class="max-w-4xl mx-auto space-y-6">
                <div class="card">
                    <div class="text-center py-12">
                        <div class="text-4xl mb-4">🔄</div>
                        <h2 class="text-xl font-semibold mb-2">FFmpeg Required</h2>
                        <p class="text-gray-400 mb-6">FFmpeg is needed for media conversion. Click below to download it automatically.</p>
                        <button id="download-ffmpeg-btn" onclick="window.downloadFFmpeg()" class="btn-primary px-8 py-3">
                            Download FFmpeg
                        </button>
                        <div id="ffmpeg-progress" class="mt-4 text-sm text-gray-400 hidden"></div>
                    </div>
                </div>
            </div>
        `;
        return;
    }

    // Load presets and recent downloads in parallel
    let presets = [];
    let recentDownloads = [];
    try {
        [presets, recentDownloads] = await Promise.all([
            App.GetConversionPresets(),
            App.GetRecentCompletedDownloads()
        ]);
    } catch (_) { /* ignore */ }

    const recentOptions = (recentDownloads || []).map(d =>
        `<option value="${escapeHtml(d.file_path)}">${escapeHtml(d.title || d.file_path)}</option>`
    ).join('');

    content.innerHTML = `
        <div class="max-w-4xl mx-auto space-y-6">
            <!-- Input File -->
            <div class="card">
                <h3 class="text-lg font-semibold mb-4">Input</h3>
                <div class="space-y-3">
                    <div class="flex space-x-2">
                        <input type="text" id="convert-input" class="input-field flex-1" placeholder="Select a media file..." readonly>
                        <button onclick="window.browseInputFile()" class="btn-secondary">Browse</button>
                    </div>
                    ${recentOptions ? `
                        <div>
                            <label class="block text-sm text-gray-400 mb-1">Or pick a recent download:</label>
                            <select id="recent-downloads" class="select-field w-full" onchange="window.selectRecentDownload(this.value)">
                                <option value="">-- Select --</option>
                                ${recentOptions}
                            </select>
                        </div>
                    ` : ''}
                </div>
            </div>

            <!-- Presets -->
            <div class="card">
                <h3 class="text-lg font-semibold mb-4">Quick Presets</h3>
                <div class="flex flex-wrap gap-2">
                    ${presets.map(p => `
                        <button onclick="window.applyConvertPreset('${p.id}')"
                                class="bg-gray-700 hover:bg-gray-600 text-sm px-3 py-2 rounded transition-colors"
                                title="${escapeHtml(p.description)}">
                            ${escapeHtml(p.name)}
                        </button>
                    `).join('')}
                </div>
            </div>

            <!-- Output Options -->
            <div class="card">
                <h3 class="text-lg font-semibold mb-4">Output Options</h3>
                <div class="space-y-4">
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label class="block text-sm font-medium mb-2">Output Format</label>
                            <select id="convert-format" class="select-field w-full">
                                <optgroup label="Video">
                                    <option value="mp4">MP4</option>
                                    <option value="mkv">MKV</option>
                                    <option value="webm">WebM</option>
                                    <option value="avi">AVI</option>
                                    <option value="mov">MOV</option>
                                </optgroup>
                                <optgroup label="Audio">
                                    <option value="mp3">MP3</option>
                                    <option value="aac">AAC</option>
                                    <option value="m4a">M4A</option>
                                    <option value="flac">FLAC</option>
                                    <option value="wav">WAV</option>
                                    <option value="ogg">OGG</option>
                                    <option value="opus">Opus</option>
                                </optgroup>
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium mb-2">Encoder Preset</label>
                            <select id="convert-preset" class="select-field w-full">
                                <option value="">Default</option>
                                <option value="ultrafast">Ultrafast</option>
                                <option value="veryfast">Very Fast</option>
                                <option value="fast">Fast</option>
                                <option value="medium">Medium</option>
                                <option value="slow">Slow (better quality)</option>
                                <option value="veryslow">Very Slow (best quality)</option>
                            </select>
                        </div>
                    </div>

                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label class="block text-sm font-medium mb-2">Video Codec</label>
                            <select id="convert-vcodec" class="select-field w-full">
                                <option value="">Auto</option>
                                <option value="libx264">H.264</option>
                                <option value="libx265">H.265 (HEVC)</option>
                                <option value="libvpx-vp9">VP9</option>
                                <option value="copy">Copy (no re-encode)</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium mb-2">Audio Codec</label>
                            <select id="convert-acodec" class="select-field w-full">
                                <option value="">Auto</option>
                                <option value="aac">AAC</option>
                                <option value="libmp3lame">MP3</option>
                                <option value="libopus">Opus</option>
                                <option value="flac">FLAC</option>
                                <option value="copy">Copy (no re-encode)</option>
                            </select>
                        </div>
                    </div>

                    <div class="grid grid-cols-3 gap-4">
                        <div>
                            <label class="block text-sm font-medium mb-2">Video Bitrate</label>
                            <input type="text" id="convert-vbitrate" class="input-field" placeholder="e.g. 5M">
                        </div>
                        <div>
                            <label class="block text-sm font-medium mb-2">Audio Bitrate</label>
                            <select id="convert-abitrate" class="select-field w-full">
                                <option value="">Default</option>
                                <option value="320k">320k</option>
                                <option value="256k">256k</option>
                                <option value="192k" selected>192k</option>
                                <option value="128k">128k</option>
                                <option value="96k">96k</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium mb-2">Resolution</label>
                            <select id="convert-resolution" class="select-field w-full">
                                <option value="">Original</option>
                                <option value="-1:2160">4K (2160p)</option>
                                <option value="-1:1440">1440p</option>
                                <option value="-1:1080">1080p</option>
                                <option value="-1:720">720p</option>
                                <option value="-1:480">480p</option>
                            </select>
                        </div>
                    </div>

                    <div>
                        <label class="block text-sm font-medium mb-2">Custom FFmpeg Arguments</label>
                        <input type="text" id="convert-custom-args" class="input-field" placeholder="e.g. -ss 00:01:00 -t 30">
                        <p class="text-xs text-gray-500 mt-1">Added to the FFmpeg command as-is. Use for trimming, filters, etc.</p>
                    </div>

                    <div>
                        <label class="block text-sm font-medium mb-2">Output File</label>
                        <div class="flex space-x-2">
                            <input type="text" id="convert-output" class="input-field flex-1" placeholder="Auto-generated from input file">
                            <button onclick="window.browseOutputFile()" class="btn-secondary">Browse</button>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Convert Button & Progress -->
            <div class="card">
                <div id="convert-action" class="space-y-4">
                    <button id="convert-start-btn" onclick="window.startConversion()" class="btn-primary w-full py-3 text-lg">
                        🔄 Convert
                    </button>
                </div>
                <div id="convert-progress" class="hidden space-y-3 mt-4">
                    <div class="flex justify-between items-center">
                        <span id="convert-status" class="text-sm text-gray-300">Converting...</span>
                        <span id="convert-speed" class="text-sm text-gray-400"></span>
                    </div>
                    <div class="progress-bar">
                        <div id="convert-progress-bar" class="progress-bar-fill" style="width: 0%"></div>
                    </div>
                    <div class="flex justify-between items-center">
                        <span id="convert-duration" class="text-sm text-gray-400"></span>
                        <button onclick="window.cancelConversion()" class="btn-danger text-sm">Cancel</button>
                    </div>
                </div>
            </div>
        </div>
    `;

    // Store presets in state for applyConvertPreset
    state.conversionPresets = presets;

    // Rehydrate UI if a conversion is already in progress
    syncConversionUI(state.conversion);
}

function syncConversionUI(job) {
    if (!job || ['completed', 'failed', 'cancelled'].includes(job.status)) return;

    document.getElementById('convert-start-btn')?.classList.add('hidden');
    document.getElementById('convert-progress')?.classList.remove('hidden');

    const bar = document.getElementById('convert-progress-bar');
    if (bar) bar.style.width = `${job.progress.toFixed(1)}%`;

    const status = document.getElementById('convert-status');
    if (status) status.textContent = `Converting... ${job.progress.toFixed(1)}%`;

    const speed = document.getElementById('convert-speed');
    if (speed) speed.textContent = job.speed || '';

    const duration = document.getElementById('convert-duration');
    if (duration) duration.textContent = job.duration || '';
}

// --- Convert Tab Handlers ---

window.downloadFFmpeg = async function() {
    const btn = document.getElementById('download-ffmpeg-btn');
    const progress = document.getElementById('ffmpeg-progress');
    if (btn) { btn.disabled = true; btn.textContent = 'Downloading...'; }
    if (progress) { progress.classList.remove('hidden'); }

    try {
        await App.DownloadFFmpeg();
        addLog('FFmpeg installed successfully');
        // Re-render to show full converter UI
        renderConvertTab();
    } catch (err) {
        addLog(`Failed to download FFmpeg: ${err}`);
        if (btn) { btn.disabled = false; btn.textContent = 'Download FFmpeg'; }
        if (progress) { progress.textContent = 'Download failed: ' + err; }
    }
};

window.browseInputFile = async function() {
    const path = await App.BrowseInputFile();
    if (path) {
        document.getElementById('convert-input').value = path;
    }
};

window.selectRecentDownload = function(path) {
    if (path) {
        document.getElementById('convert-input').value = path;
    }
};

window.browseOutputFile = async function() {
    const inputPath = document.getElementById('convert-input')?.value || '';
    const format = document.getElementById('convert-format')?.value || 'mp4';
    let defaultName = 'output.' + format;
    if (inputPath) {
        const base = inputPath.replace(/\.[^.]+$/, '');
        defaultName = base.split(/[\\/]/).pop() + '_converted.' + format;
    }
    const path = await App.BrowseOutputFile(defaultName);
    if (path) {
        document.getElementById('convert-output').value = path;
    }
};

window.applyConvertPreset = function(presetId) {
    const preset = (state.conversionPresets || []).find(p => p.id === presetId);
    if (!preset) return;

    const setField = (id, key) => {
        if (!(key in preset)) return;
        const el = document.getElementById(id);
        if (el) el.value = preset[key] ?? '';
    };

    setField('convert-format', 'output_format');
    setField('convert-vcodec', 'video_codec');
    setField('convert-acodec', 'audio_codec');
    setField('convert-preset', 'preset');
    setField('convert-abitrate', 'audio_bitrate');

    addVerboseLog(`Applied preset: ${preset.name}`);
};

window.startConversion = async function() {
    const inputFile = document.getElementById('convert-input')?.value;
    if (!inputFile) {
        alert('Please select an input file');
        return;
    }

    const opts = {
        input_file:    inputFile,
        output_file:   document.getElementById('convert-output')?.value || '',
        output_format: document.getElementById('convert-format')?.value || 'mp4',
        video_codec:   document.getElementById('convert-vcodec')?.value || '',
        audio_codec:   document.getElementById('convert-acodec')?.value || '',
        preset:        document.getElementById('convert-preset')?.value || '',
        video_bitrate: document.getElementById('convert-vbitrate')?.value || '',
        audio_bitrate: document.getElementById('convert-abitrate')?.value || '',
        resolution:    document.getElementById('convert-resolution')?.value || '',
        custom_args:   document.getElementById('convert-custom-args')?.value || ''
    };

    try {
        await App.StartConversion(opts);
        addLog(`Converting: ${inputFile}`);

        // Show progress, hide start button
        const startBtn = document.getElementById('convert-start-btn');
        if (startBtn) startBtn.classList.add('hidden');
        const progressDiv = document.getElementById('convert-progress');
        if (progressDiv) progressDiv.classList.remove('hidden');
    } catch (err) {
        alert('Failed to start conversion: ' + err);
        addLog(`Conversion error: ${err}`);
    }
};

window.cancelConversion = async function() {
    try {
        await App.CancelConversion();
        addLog('Conversion cancelled');
        resetConvertUI();
    } catch (err) {
        addLog(`Cancel error: ${err}`);
    }
};

window.resetConvertUI = resetConvertUI;
function resetConvertUI() {
    const startBtn = document.getElementById('convert-start-btn');
    if (startBtn) startBtn.classList.remove('hidden');
    const progressDiv = document.getElementById('convert-progress');
    if (progressDiv) progressDiv.classList.add('hidden');
    const resultDiv = document.getElementById('convert-result');
    if (resultDiv) resultDiv.remove();
    const bar = document.getElementById('convert-progress-bar');
    if (bar) bar.style.width = '0%';
    const status = document.getElementById('convert-status');
    if (status) {
        status.textContent = '';
        status.classList.remove('text-green-400', 'text-red-400');
    }
    const speed = document.getElementById('convert-speed');
    if (speed) speed.textContent = '';
    const duration = document.getElementById('convert-duration');
    if (duration) duration.textContent = '';
}

function showConvertResult(status, detail) {
    const progressDiv = document.getElementById('convert-progress');
    if (progressDiv) progressDiv.classList.add('hidden');
    const startBtn = document.getElementById('convert-start-btn');
    if (startBtn) startBtn.classList.add('hidden');

    // Remove any previous result
    const prev = document.getElementById('convert-result');
    if (prev) prev.remove();

    const container = document.getElementById('convert-action');
    if (!container) return;

    const isSuccess = status === 'completed';
    const resultHtml = isSuccess
        ? `<div id="convert-result" class="space-y-3">
               <div class="flex items-center gap-2 text-green-400 font-medium">
                   <span>✓</span> Conversion complete!
               </div>
               <div class="text-sm text-gray-400 truncate" title="${escapeHtml(detail)}">${escapeHtml(detail)}</div>
               <div class="flex gap-2">
                   <button onclick="window.openConvertOutput()" class="btn-secondary flex-1">
                       📂 Open in Folder
                   </button>
                   <button onclick="window.resetConvertUI()" class="btn-primary flex-1">
                       🔄 Convert Another
                   </button>
               </div>
           </div>`
        : `<div id="convert-result" class="space-y-3">
               <div class="flex items-center gap-2 text-red-400 font-medium">
                   <span>✕</span> Conversion failed
               </div>
               <div class="text-sm text-gray-400 bg-gray-800/50 rounded p-2 font-mono break-all">${escapeHtml(detail)}</div>
               <button onclick="window.resetConvertUI()" class="btn-primary w-full">
                   Try Again
               </button>
           </div>`;

    container.insertAdjacentHTML('beforeend', resultHtml);

    if (isSuccess) {
        state._lastConvertOutput = detail;
    }
}

window.openConvertOutput = function() {
    if (state._lastConvertOutput) {
        App.OpenFileInFolder(state._lastConvertOutput).catch(err => {
            addLog(`Failed to open output folder: ${err}`);
        });
    }
};

// History Tab
function renderHistoryTab() {
    const content = document.getElementById('tab-content');
    content.innerHTML = `
        <div class="max-w-6xl mx-auto space-y-6">
            <div class="flex justify-between items-center">
                <h2 class="text-2xl font-semibold">Download History</h2>
                <div class="flex space-x-2">
                    <button onclick="window.refreshHistory()" class="btn-secondary">
                        🔄 Refresh
                    </button>
                    <button onclick="window.clearHistory()" class="btn-danger">
                        🗑️ Clear All
                    </button>
                </div>
            </div>

            <div class="card">
                <input
                    type="text"
                    id="history-search"
                    class="input-field mb-4"
                    placeholder="Search by title or URL..."
                    oninput="window.filterHistory()"
                />

                <div id="history-list" class="space-y-2 max-h-[600px] overflow-y-auto">
                    <p class="text-gray-400 text-center py-8">Loading...</p>
                </div>
            </div>
        </div>
    `;

    refreshHistory();
}

window.refreshHistory = async function() {
    try {
        state.history = await App.GetHistory("", "") || [];
        displayHistory();
    } catch (err) {
        console.error('Failed to load history:', err);
        const list = document.getElementById('history-list');
        if (list) {
            list.innerHTML = '<p class="text-red-400 text-center py-8">Failed to load history</p>';
        }
    }
};

window.filterHistory = function() {
    displayHistory();
};

function displayHistory() {
    const list = document.getElementById('history-list');
    if (!list) return;

    const search = document.getElementById('history-search')?.value.toLowerCase() || '';
    const filtered = state.history.filter(item =>
        (item.title || '').toLowerCase().includes(search) ||
        (item.url || '').toLowerCase().includes(search)
    );

    if (filtered.length === 0) {
        list.innerHTML = '<p class="text-gray-400 text-center py-8">No history items found</p>';
        return;
    }

    list.innerHTML = filtered.map(item => {
        const hasFile = item.file_path && item.file_exists !== false;
        const fileMissing = item.file_path && item.file_exists === false;
        return `
        <div class="bg-gray-700 rounded p-3 flex justify-between items-center hover:bg-gray-600 transition-colors${fileMissing ? ' opacity-50' : ''}">
            <div class="flex-1 min-w-0">
                <div class="font-medium truncate${hasFile ? ' cursor-pointer hover:text-blue-400' : ''}"
                     ${hasFile ? `onclick="window.openFileInFolder('${escapeJsStr(item.file_path)}')"` : ''}>
                    ${escapeHtml(item.title || item.url)}
                </div>
                ${item.title ? `
                    <div class="text-sm text-gray-400 truncate cursor-pointer hover:text-blue-400"
                         onclick="window.openURL('${escapeJsStr(item.url)}')">
                        ${escapeHtml(item.url)}
                    </div>
                ` : ''}
                <div class="text-xs text-gray-500 mt-1">
                    ${new Date(item.date).toLocaleString()} • ${item.status}${fileMissing ? ' • <span class="text-red-400 italic">File missing</span>' : ''}
                </div>
            </div>
            <div class="flex items-center space-x-2 shrink-0 ml-4">
                <button onclick="window.redownload('${escapeJsStr(item.url)}')" class="btn-secondary text-sm">
                    ⬇️ Re-download
                </button>
                <button onclick="window.removeHistoryEntry('${item.id}')" class="text-gray-500 hover:text-red-400 text-lg leading-none" title="Delete">🗑️</button>
            </div>
        </div>`;
    }).join('');
}

window.clearHistory = async function() {
    if (!confirm('Are you sure you want to clear all history?')) return;

    try {
        await App.ClearHistory();
        state.history = [];
        displayHistory();
    } catch (err) {
        alert('Failed to clear history: ' + err);
    }
};

window.hideFromQueue = async function(id) {
    try {
        await App.HideFromQueue(id);
        state.recentHistory = state.recentHistory.filter(h => h.id !== id);
        if (state.currentTab === 'download') {
            updateDownloadQueue();
        }
    } catch (err) {
        addLog(`Failed to hide from queue: ${err}`);
    }
};

window.removeHistoryEntry = async function(id) {
    try {
        await App.RemoveHistoryEntry(id);
        state.history = state.history.filter(h => h.id !== id);
        displayHistory();
    } catch (err) {
        addLog(`Failed to remove history entry: ${err}`);
    }
};

window.openFile = function(path) {
    if (path) {
        App.OpenFile(path).catch(err => console.error('Failed to open file:', err));
    }
};

window.openFileInFolder = async function(path) {
    if (!path) {
        addLog(`⚠️ file_path is empty — cannot open folder`);
        return;
    }
    try {
        await App.OpenFileInFolder(path);
    } catch (err) {
        addLog(`❌ OpenFileInFolder error: ${err}`);
    }
};

window.openURL = function(url) {
    if (url) {
        App.OpenURL(url).catch(err => console.error('Failed to open URL:', err));
    }
};

window.redownload = function(url) {
    state.currentTab = 'download';
    switchTab('download');
    setTimeout(() => {
        const input = document.getElementById('url-input');
        if (input) {
            input.value = url;
            input.focus();
        }
    }, 100);
};

// Settings Tab
function renderSettingsTab() {
    const content = document.getElementById('tab-content');
    content.innerHTML = `
        <div class="max-w-4xl mx-auto space-y-6">
            <div class="flex justify-between items-center">
                <h2 class="text-2xl font-semibold">Settings</h2>
                <button onclick="window.saveSettings()" class="btn-primary">
                    💾 Save Settings
                </button>
            </div>

            <div id="settings-content" class="space-y-6">
                <p class="text-gray-400">Loading settings...</p>
            </div>
        </div>
    `;

    loadSettingsContent();
}

async function loadSettings() {
    try {
        state.settings = await App.GetSettings();
        addVerboseLog('Settings loaded');
    } catch (err) {
        console.error('Failed to load settings:', err);
        addLog('⚠️ Failed to load settings: ' + err);
    }
}

async function loadSettingsContent() {
    await loadSettings();
    const container = document.getElementById('settings-content');
    if (!container || !state.settings) return;

    const s = state.settings;

    container.innerHTML = `
        <!-- General Settings -->
        <div class="card">
            <h3 class="text-lg font-semibold mb-4">General</h3>
            <div class="space-y-4">
                <div>
                    <label class="block text-sm font-medium mb-2">Save Folder</label>
                    <div class="flex space-x-2">
                        <input type="text" id="save-folder" value="${escapeHtml(s.General.SaveFolder)}"
                               class="input-field flex-1" readonly>
                        <button onclick="window.selectFolder()" class="btn-secondary">Browse</button>
                    </div>
                </div>

                <div>
                    <label class="block text-sm font-medium mb-2">Max Concurrent Downloads</label>
                    <input type="number" id="max-concurrent" value="${s.General.MaxConcurrentDownloads}"
                           min="1" max="10" class="input-field">
                </div>

                <div class="space-y-2">
                    <label class="flex items-center space-x-2 cursor-pointer">
                        <input type="checkbox" id="check-updates" ${s.General.CheckUpdatesOnStart ? 'checked' : ''}
                               class="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600">
                        <span class="text-sm">Check for updates on startup</span>
                    </label>
                    <label class="flex items-center space-x-2 cursor-pointer">
                        <input type="checkbox" id="clipboard-monitoring" ${s.General.ClipboardMonitoring ? 'checked' : ''}
                               class="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600">
                        <span class="text-sm">Monitor clipboard for URLs</span>
                    </label>
                    <label class="flex items-center space-x-2 cursor-pointer">
                        <input type="checkbox" id="notifications" ${s.General.NotificationsEnabled ? 'checked' : ''}
                               class="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600">
                        <span class="text-sm">Enable notifications</span>
                    </label>
                    <label class="flex items-center space-x-2 cursor-pointer">
                        <input type="checkbox" id="verbose-logging" ${s.General.VerboseLogging ? 'checked' : ''}
                               class="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600">
                        <span class="text-sm">Verbose logging</span>
                    </label>
                </div>
            </div>
        </div>

        <!-- Download Settings -->
        <div class="card">
            <h3 class="text-lg font-semibold mb-4">Download Defaults</h3>
            <div class="space-y-4">
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label class="block text-sm font-medium mb-2">Audio Format</label>
                        <select id="audio-format" class="select-field w-full">
                            <option value="mp3" ${s.Download.AudioFormat === 'mp3' ? 'selected' : ''}>MP3</option>
                            <option value="m4a" ${s.Download.AudioFormat === 'm4a' ? 'selected' : ''}>M4A</option>
                            <option value="opus" ${s.Download.AudioFormat === 'opus' ? 'selected' : ''}>Opus</option>
                            <option value="flac" ${s.Download.AudioFormat === 'flac' ? 'selected' : ''}>FLAC</option>
                        </select>
                    </div>
                    <div>
                        <label class="block text-sm font-medium mb-2">Audio Quality</label>
                        <select id="audio-quality" class="select-field w-full">
                            <option value="320" ${s.Download.AudioQuality === '320' ? 'selected' : ''}>320k</option>
                            <option value="256" ${s.Download.AudioQuality === '256' ? 'selected' : ''}>256k</option>
                            <option value="192" ${s.Download.AudioQuality === '192' ? 'selected' : ''}>192k</option>
                            <option value="128" ${s.Download.AudioQuality === '128' ? 'selected' : ''}>128k</option>
                        </select>
                    </div>
                </div>

                <div class="space-y-2">
                    <label class="flex items-center space-x-2 cursor-pointer">
                        <input type="checkbox" id="embed-thumbnail" ${s.Download.EmbedThumbnail ? 'checked' : ''}
                               class="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600">
                        <span class="text-sm">Embed thumbnail</span>
                    </label>
                    <label class="flex items-center space-x-2 cursor-pointer">
                        <input type="checkbox" id="embed-metadata" ${s.Download.EmbedMetadata ? 'checked' : ''}
                               class="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600">
                        <span class="text-sm">Embed metadata</span>
                    </label>
                    <label class="flex items-center space-x-2 cursor-pointer">
                        <input type="checkbox" id="embed-chapters" ${s.Download.EmbedChapters ? 'checked' : ''}
                               class="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600">
                        <span class="text-sm">Embed chapters</span>
                    </label>
                    <label class="flex items-center space-x-2 cursor-pointer">
                        <input type="checkbox" id="sponsorblock" ${s.Download.Sponsorblock ? 'checked' : ''}
                               class="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600">
                        <span class="text-sm">Remove sponsored segments (SponsorBlock)</span>
                    </label>
                </div>
            </div>
        </div>

        <!-- Network Settings -->
        <div class="card">
            <h3 class="text-lg font-semibold mb-4">Network</h3>
            <div class="space-y-4">
                <div>
                    <label class="block text-sm font-medium mb-2">Rate Limit (e.g., 50K, 4.2M)</label>
                    <input type="text" id="rate-limit" value="${escapeHtml(s.Network.RateLimit)}"
                           class="input-field" placeholder="No limit">
                </div>
                <div>
                    <label class="block text-sm font-medium mb-2">Proxy</label>
                    <input type="text" id="proxy" value="${escapeHtml(s.Network.Proxy)}"
                           class="input-field" placeholder="http://proxy:port">
                </div>
                <div>
                    <label class="block text-sm font-medium mb-2">Retries</label>
                    <input type="number" id="retries" value="${s.Network.Retries}"
                           min="0" max="100" class="input-field">
                </div>
            </div>
        </div>

        <!-- Authentication Settings -->
        <div class="card">
            <h3 class="text-lg font-semibold mb-4">Authentication</h3>
            <div class="space-y-4">
                <!-- Account Risk Warning -->
                <div class="bg-amber-900/30 border border-amber-600/50 rounded-lg p-3">
                    <div class="flex items-start gap-2">
                        <span class="text-amber-400 mt-0.5">⚠️</span>
                        <div class="text-sm">
                            <div class="font-medium text-amber-300 mb-1">Account Risk Warning</div>
                            <p class="text-gray-400">
                                Using cookies or authentication may put your YouTube account at risk of being flagged or banned.
                                <strong class="text-amber-200">Consider using a throwaway account</strong> rather than your main account.
                                Excessive downloading can trigger YouTube's anti-bot measures.
                            </p>
                        </div>
                    </div>
                </div>

                <div>
                    <label class="block text-sm font-medium mb-2">Cookies from Browser</label>
                    <select id="cookies-browser" class="select-field w-full">
                        <option value="" ${!s.Auth.CookiesBrowser ? 'selected' : ''}>None</option>
                        <option value="chrome" ${s.Auth.CookiesBrowser === 'chrome' ? 'selected' : ''}>Chrome</option>
                        <option value="firefox" ${s.Auth.CookiesBrowser === 'firefox' ? 'selected' : ''}>Firefox</option>
                        <option value="edge" ${s.Auth.CookiesBrowser === 'edge' ? 'selected' : ''}>Edge</option>
                        <option value="brave" ${s.Auth.CookiesBrowser === 'brave' ? 'selected' : ''}>Brave</option>
                        <option value="opera" ${s.Auth.CookiesBrowser === 'opera' ? 'selected' : ''}>Opera</option>
                    </select>
                </div>
                <div>
                    <label class="block text-sm font-medium mb-2">Cookies File Path</label>
                    <input type="text" id="cookies-file" value="${escapeHtml(s.Auth.CookiesFile)}"
                           class="input-field" placeholder="Path to cookies.txt">
                </div>
                <div>
                    <label class="block text-sm font-medium mb-2">YouTube Player Client</label>
                    <select id="player-client" class="select-field w-full">
                        <option value="default" ${!s.Auth.PlayerClient || s.Auth.PlayerClient === 'default' ? 'selected' : ''}>Default (Auto)</option>
                        <option value="mweb" ${s.Auth.PlayerClient === 'mweb' ? 'selected' : ''}>Mobile Web (Recommended for 403 errors)</option>
                        <option value="web_creator" ${s.Auth.PlayerClient === 'web_creator' ? 'selected' : ''}>Web Creator</option>
                        <option value="ios" ${s.Auth.PlayerClient === 'ios' ? 'selected' : ''}>iOS</option>
                        <option value="android" ${s.Auth.PlayerClient === 'android' ? 'selected' : ''}>Android</option>
                    </select>
                    <p class="text-xs text-gray-500 mt-1">Try "Mobile Web" if you get HTTP 403 errors</p>
                </div>
                <div>
                    <label class="block text-sm font-medium mb-2">PO Token (YouTube)</label>
                    <input type="text" id="po-token" value="${escapeHtml(s.Auth.POToken)}"
                           class="input-field" placeholder="Paste your PO Token here">
                </div>

                <!-- PO Token Guide -->
                <details class="bg-gray-800/50 rounded-lg border border-gray-700">
                    <summary class="px-4 py-3 cursor-pointer text-sm font-medium text-blue-400 hover:text-blue-300 flex items-center gap-2">
                        <span>🔑</span> How to Get a PO Token
                    </summary>
                    <div class="px-4 pb-4 space-y-3 text-sm">
                        <p class="text-gray-400">
                            PO Tokens help download age-restricted or blocked videos by proving you're a real user.
                        </p>

                        <div class="bg-green-900/30 border border-green-700/50 rounded p-3">
                            <div class="font-medium text-green-400 mb-2">Option A: Browser Extension (Easiest)</div>
                            <p class="text-gray-400 mb-2">Install a plugin that automatically generates PO Tokens:</p>
                            <button
                                onclick="window.openURL('https://github.com/Brainicism/bgutil-ytdlp-pot-provider')"
                                class="bg-green-700 hover:bg-green-600 text-white text-xs px-3 py-1.5 rounded"
                            >
                                Get bgutil-ytdlp-pot-provider
                            </button>
                        </div>

                        <div class="bg-gray-700/50 rounded p-3">
                            <div class="font-medium text-gray-300 mb-2">Option B: Manual Extraction</div>
                            <ol class="text-gray-400 space-y-1 list-decimal list-inside">
                                <li>Open <span class="text-blue-400">music.youtube.com</span> in your browser</li>
                                <li>Press <kbd class="bg-gray-600 px-1 rounded">F12</kbd> to open Developer Tools</li>
                                <li>Go to <span class="text-yellow-400">Network</span> tab, filter by <code class="bg-gray-600 px-1 rounded">player</code></li>
                                <li>Play any video, find the <code class="bg-gray-600 px-1 rounded">player</code> request</li>
                                <li>In the request payload, find <code class="bg-gray-600 px-1 rounded">serviceIntegrityDimensions.poToken</code></li>
                                <li>Copy the token value and paste it above</li>
                            </ol>
                        </div>

                        <button
                            onclick="window.openURL('https://github.com/yt-dlp/yt-dlp/wiki/PO-Token-Guide')"
                            class="text-blue-400 hover:text-blue-300 text-xs flex items-center gap-1"
                        >
                            📖 View Full Guide on GitHub
                        </button>
                    </div>
                </details>
            </div>
        </div>

        <!-- Advanced Settings -->
        <div class="card">
            <h3 class="text-lg font-semibold mb-4">Advanced</h3>
            <div class="space-y-4">
                <div>
                    <label class="flex items-center space-x-2 cursor-pointer">
                        <input type="checkbox" id="use-nightly" ${s.Advanced.UseNightly ? 'checked' : ''}
                               class="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600">
                        <span class="text-sm">Use nightly builds (experimental features)</span>
                    </label>
                </div>
                <div>
                    <label class="block text-sm font-medium mb-2">Output Template</label>
                    <input type="text" id="output-template" value="${escapeHtml(s.Advanced.OutputTemplate)}"
                           class="input-field" placeholder="%(title)s.%(ext)s">
                </div>
                <div class="relative" id="extra-args-wrapper">
                    <label class="block text-sm font-medium mb-2">Extra Arguments</label>
                    <input type="text" id="extra-args" value="${escapeHtml(s.Advanced.ExtraArgs)}"
                           class="input-field" placeholder="--verbose --ignore-errors" autocomplete="off">
                    <div id="extra-args-dropdown" class="hidden absolute z-50 w-full mt-1 bg-gray-700 border border-gray-600 rounded-lg shadow-lg max-h-48 overflow-y-auto"></div>
                    <p class="text-xs text-gray-500 mt-1">Type -- to see suggestions. Keeps suggesting as you add more arguments.</p>
                </div>
                <div>
                    <button onclick="window.checkForUpdates()" class="btn-secondary w-full">
                        🔄 Check for yt-dlp Updates
                    </button>
                </div>
            </div>
        </div>

        <!-- JavaScript Runtime -->
        <div class="card">
            <h3 class="text-lg font-semibold mb-4">JavaScript Runtime</h3>
            <p class="text-sm text-gray-400 mb-4">
                Some YouTube videos require JavaScript execution. yt-dlp supports Deno, Node.js, and Bun.
            </p>
            <div id="js-runtime-status" class="space-y-4">
                <div class="text-gray-400 text-sm">Detecting runtimes...</div>
            </div>
        </div>
    `;

    // Setup auto-save and custom autocomplete after DOM is ready
    setupSettingsAutoSave();
    setupExtraArgsAutocomplete();

    // Load JS runtime info asynchronously
    loadJSRuntimeStatus();
}

// Load and display JS runtime status
async function loadJSRuntimeStatus() {
    const container = document.getElementById('js-runtime-status');
    if (!container) return;

    try {
        const info = await App.GetJSRuntimeInfo();
        const s = state.settings;

        let statusHtml = '';

        if (info.detected) {
            statusHtml += `
                <div class="bg-green-900/30 border border-green-700/50 rounded-lg p-3">
                    <div class="flex items-center gap-2">
                        <span class="text-green-400">✓</span>
                        <span class="font-medium text-green-300">Runtime detected: ${escapeHtml(info.detected.name)}</span>
                    </div>
                    <div class="text-sm text-gray-400 mt-1">
                        Version: ${escapeHtml(info.detected.version)}<br>
                        Path: <span class="font-mono text-xs">${escapeHtml(info.detected.path)}</span>
                    </div>
                </div>
            `;
        } else {
            statusHtml += `
                <div class="bg-yellow-900/30 border border-yellow-700/50 rounded-lg p-3">
                    <div class="flex items-center gap-2">
                        <span class="text-yellow-400">⚠️</span>
                        <span class="font-medium text-yellow-300">No JavaScript runtime detected</span>
                    </div>
                    <div class="text-sm text-gray-400 mt-1">
                        Some YouTube videos may fail to download without a JS runtime.
                    </div>
                </div>
            `;
        }

        // Runtime selector
        statusHtml += `
            <div>
                <label class="block text-sm font-medium mb-2">Runtime Selection</label>
                <select id="js-runtime" class="select-field w-full">
                    <option value="auto" ${!s.Advanced.JSRuntime || s.Advanced.JSRuntime === 'auto' ? 'selected' : ''}>Auto-detect (Recommended)</option>
                    <option value="deno" ${s.Advanced.JSRuntime === 'deno' ? 'selected' : ''}>Deno</option>
                    <option value="node" ${s.Advanced.JSRuntime === 'node' ? 'selected' : ''}>Node.js</option>
                    <option value="bun" ${s.Advanced.JSRuntime === 'bun' ? 'selected' : ''}>Bun</option>
                </select>
                <p class="text-xs text-gray-500 mt-1">Choose which runtime yt-dlp should use for JavaScript execution</p>
            </div>
        `;

        // Available runtimes list
        if (info.available && info.available.length > 0) {
            statusHtml += `
                <details class="text-sm">
                    <summary class="text-gray-500 cursor-pointer hover:text-gray-400">
                        Available runtimes (${info.available.length})
                    </summary>
                    <div class="mt-2 space-y-1">
                        ${info.available.map(r => `
                            <div class="flex justify-between text-gray-400 bg-gray-800/50 rounded px-2 py-1">
                                <span>${escapeHtml(r.name)} ${escapeHtml(r.version)}</span>
                                <span class="font-mono text-xs truncate max-w-xs">${escapeHtml(r.path)}</span>
                            </div>
                        `).join('')}
                    </div>
                </details>
            `;
        }

        // Download Deno button
        statusHtml += `
            <div class="pt-2 border-t border-gray-700">
                <button id="download-deno-btn" onclick="window.downloadDeno()" class="btn-secondary w-full">
                    ⬇️ Download Bundled Deno
                </button>
                <p class="text-xs text-gray-500 mt-1 text-center">
                    Downloads Deno to the app data folder (recommended by yt-dlp team)
                </p>
            </div>
        `;

        container.innerHTML = statusHtml;

        // Add event listener for the runtime selector
        const runtimeSelect = document.getElementById('js-runtime');
        if (runtimeSelect) {
            runtimeSelect.addEventListener('change', autoSaveSettings);
        }

    } catch (err) {
        container.innerHTML = `
            <div class="text-red-400 text-sm">Failed to detect runtimes: ${escapeHtml(err.toString())}</div>
        `;
    }
}

// Debounce timer for auto-save
let autoSaveTimer = null;

// Collect settings from form elements
function collectSettingsFromForm() {
    if (!state.settings) return false;

    const getElement = (id) => document.getElementById(id);

    // Only collect if elements exist (we're on settings tab)
    const maxConcurrent = getElement('max-concurrent');
    if (!maxConcurrent) return false;

    state.settings.General.MaxConcurrentDownloads = parseInt(maxConcurrent.value) || 3;
    state.settings.General.CheckUpdatesOnStart = getElement('check-updates')?.checked ?? state.settings.General.CheckUpdatesOnStart;
    state.settings.General.ClipboardMonitoring = getElement('clipboard-monitoring')?.checked ?? state.settings.General.ClipboardMonitoring;
    state.settings.General.NotificationsEnabled = getElement('notifications')?.checked ?? state.settings.General.NotificationsEnabled;
    state.settings.General.VerboseLogging = getElement('verbose-logging')?.checked ?? state.settings.General.VerboseLogging;

    state.settings.Download.AudioFormat = getElement('audio-format')?.value ?? state.settings.Download.AudioFormat;
    state.settings.Download.AudioQuality = getElement('audio-quality')?.value ?? state.settings.Download.AudioQuality;
    state.settings.Download.EmbedThumbnail = getElement('embed-thumbnail')?.checked ?? state.settings.Download.EmbedThumbnail;
    state.settings.Download.EmbedMetadata = getElement('embed-metadata')?.checked ?? state.settings.Download.EmbedMetadata;
    state.settings.Download.EmbedChapters = getElement('embed-chapters')?.checked ?? state.settings.Download.EmbedChapters;
    state.settings.Download.Sponsorblock = getElement('sponsorblock')?.checked ?? state.settings.Download.Sponsorblock;

    state.settings.Network.RateLimit = getElement('rate-limit')?.value ?? state.settings.Network.RateLimit;
    state.settings.Network.Proxy = getElement('proxy')?.value ?? state.settings.Network.Proxy;
    state.settings.Network.Retries = parseInt(getElement('retries')?.value) || 10;

    state.settings.Auth.CookiesBrowser = getElement('cookies-browser')?.value ?? state.settings.Auth.CookiesBrowser;
    state.settings.Auth.CookiesFile = getElement('cookies-file')?.value ?? state.settings.Auth.CookiesFile;
    state.settings.Auth.PlayerClient = getElement('player-client')?.value ?? state.settings.Auth.PlayerClient;
    state.settings.Auth.POToken = getElement('po-token')?.value ?? state.settings.Auth.POToken;

    state.settings.Advanced.UseNightly = getElement('use-nightly')?.checked ?? state.settings.Advanced.UseNightly;
    state.settings.Advanced.OutputTemplate = getElement('output-template')?.value ?? state.settings.Advanced.OutputTemplate;
    state.settings.Advanced.ExtraArgs = getElement('extra-args')?.value ?? state.settings.Advanced.ExtraArgs;
    state.settings.Advanced.JSRuntime = getElement('js-runtime')?.value ?? state.settings.Advanced.JSRuntime;

    return true;
}

// Auto-save settings with debouncing (waits 500ms after last change)
function autoSaveSettings() {
    if (autoSaveTimer) {
        clearTimeout(autoSaveTimer);
    }

    autoSaveTimer = setTimeout(async () => {
        if (!collectSettingsFromForm()) return;

        try {
            await App.SaveSettings(state.settings);
            addVerboseLog('Settings auto-saved');
        } catch (err) {
            console.error('Auto-save failed:', err);
        }
    }, 500);
}

// Setup auto-save listeners for settings inputs
function setupSettingsAutoSave() {
    const settingsInputIds = [
        'max-concurrent', 'check-updates', 'clipboard-monitoring', 'notifications', 'verbose-logging',
        'audio-format', 'audio-quality', 'embed-thumbnail', 'embed-metadata',
        'embed-chapters', 'sponsorblock', 'rate-limit', 'proxy', 'retries',
        'cookies-browser', 'cookies-file', 'player-client', 'po-token', 'use-nightly',
        'output-template', 'extra-args'
    ];

    settingsInputIds.forEach(id => {
        const element = document.getElementById(id);
        if (element) {
            const eventType = element.type === 'checkbox' ? 'change' : 'input';
            element.addEventListener(eventType, autoSaveSettings);
        }
    });
}

const YTDLP_ARG_SUGGESTIONS = [
    { value: '--verbose', desc: 'Enable verbose output' },
    { value: '--ignore-errors', desc: 'Skip unavailable videos in playlist' },
    { value: '--no-playlist', desc: 'Download single video even if URL contains playlist' },
    { value: '--yes-playlist', desc: 'Download entire playlist' },
    { value: '--flat-playlist', desc: 'List playlist contents without downloading' },
    { value: '--no-check-certificates', desc: 'Ignore SSL certificate errors' },
    { value: '--prefer-free-formats', desc: 'Prefer free codecs (webm, opus, vp9)' },
    { value: '--no-mtime', desc: "Don't use Last-modified header for file timestamp" },
    { value: '--write-subs', desc: 'Download subtitles' },
    { value: '--write-auto-subs', desc: 'Download auto-generated subtitles' },
    { value: '--sub-langs all', desc: 'Download all available subtitle languages' },
    { value: '--write-description', desc: 'Write video description to file' },
    { value: '--write-info-json', desc: 'Write video metadata to JSON file' },
    { value: '--write-comments', desc: 'Download video comments' },
    { value: '--no-overwrites', desc: "Don't overwrite existing files" },
    { value: '--restrict-filenames', desc: 'Restrict filenames to ASCII characters' },
    { value: '--trim-filenames 100', desc: 'Limit filename length to 100 chars' },
    { value: '--extract-audio', desc: 'Extract audio only (convert if needed)' },
    { value: '--keep-video', desc: 'Keep video file after audio extraction' },
    { value: '--remux-video mp4', desc: 'Remux to mp4 container' },
    { value: '--recode-video mp4', desc: 'Re-encode to mp4' },
    { value: '--sleep-interval 5', desc: 'Wait 5 seconds between downloads' },
    { value: '--max-sleep-interval 30', desc: 'Random sleep up to 30 seconds' },
    { value: '--concurrent-fragments 4', desc: 'Download 4 fragments in parallel' },
    { value: '--geo-bypass', desc: 'Bypass geographic restrictions' },
    { value: '--force-ipv4', desc: 'Force IPv4 connection' },
    { value: '--force-ipv6', desc: 'Force IPv6 connection' },
    { value: '--match-filter !is_live', desc: 'Skip live streams' },
    { value: '--download-archive archive.txt', desc: 'Track downloaded videos' },
    { value: '--print-to-file filename %(title)s titles.txt', desc: 'Save titles to file' },
];

// Custom multi-token autocomplete — matches the last argument being typed
function setupExtraArgsAutocomplete() {
    const input = document.getElementById('extra-args');
    const dropdown = document.getElementById('extra-args-dropdown');
    if (!input || !dropdown) return;

    let selectedIndex = -1;

    function getLastToken(text) {
        const trimmed = text.trimEnd();
        // Find the start of the last --flag group
        const lastDashDash = trimmed.lastIndexOf(' --');
        if (lastDashDash !== -1) return trimmed.slice(lastDashDash + 1);
        // If text starts with -- and has no prior space-delimited flags
        if (trimmed.startsWith('--')) return trimmed;
        return '';
    }

    function showSuggestions() {
        const token = getLastToken(input.value);
        if (token.length < 2) {
            dropdown.classList.add('hidden');
            return;
        }

        const tokenLower = token.toLowerCase();
        const alreadyUsed = input.value.toLowerCase();
        const matches = YTDLP_ARG_SUGGESTIONS.filter(s => {
            // Match against the flag portion of the suggestion
            const flag = s.value.split(' ')[0];
            return flag.toLowerCase().includes(tokenLower.split(' ')[0]) && !alreadyUsed.includes(flag);
        });

        if (matches.length === 0) {
            dropdown.classList.add('hidden');
            return;
        }

        selectedIndex = -1;
        dropdown.innerHTML = matches.map((s, i) => `
            <div class="px-3 py-2 cursor-pointer hover:bg-gray-600 text-sm flex justify-between items-center" data-index="${i}" data-value="${escapeHtml(s.value)}">
                <span class="font-mono text-blue-300">${escapeHtml(s.value)}</span>
                <span class="text-gray-400 text-xs ml-3 truncate">${escapeHtml(s.desc)}</span>
            </div>
        `).join('');
        dropdown.classList.remove('hidden');
    }

    function pickSuggestion(value) {
        const text = input.value;
        const trimmed = text.trimEnd();
        const lastDashDash = trimmed.lastIndexOf(' --');
        if (lastDashDash !== -1) {
            input.value = trimmed.slice(0, lastDashDash + 1) + value + ' ';
        } else {
            // Replace entire input if it was the first argument
            const prefix = trimmed.startsWith('--') ? '' : trimmed;
            input.value = (prefix ? prefix + ' ' : '') + value + ' ';
        }
        dropdown.classList.add('hidden');
        input.focus();
        autoSaveSettings();
    }

    input.addEventListener('input', showSuggestions);
    input.addEventListener('focus', showSuggestions);

    input.addEventListener('keydown', (e) => {
        if (dropdown.classList.contains('hidden')) return;
        const items = dropdown.children;
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            selectedIndex = Math.min(selectedIndex + 1, items.length - 1);
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            selectedIndex = Math.max(selectedIndex - 1, 0);
        } else if (e.key === 'Enter' && selectedIndex >= 0) {
            e.preventDefault();
            pickSuggestion(items[selectedIndex].dataset.value);
            return;
        } else if (e.key === 'Escape') {
            dropdown.classList.add('hidden');
            return;
        } else {
            return;
        }
        for (let i = 0; i < items.length; i++) {
            items[i].classList.toggle('bg-gray-600', i === selectedIndex);
        }
    });

    dropdown.addEventListener('mousedown', (e) => {
        // mousedown instead of click so it fires before blur
        const item = e.target.closest('[data-value]');
        if (item) pickSuggestion(item.dataset.value);
    });

    input.addEventListener('blur', () => {
        // Small delay so mousedown on dropdown fires first
        setTimeout(() => dropdown.classList.add('hidden'), 150);
    });
}

window.saveSettings = async function() {
    if (!collectSettingsFromForm()) return;

    try {
        await App.SaveSettings(state.settings);
        alert('Settings saved successfully!');
    } catch (err) {
        alert('Failed to save settings: ' + err);
    }
};


window.downloadDeno = async function() {
    const btn = document.getElementById('download-deno-btn');
    if (btn) {
        btn.disabled = true;
        btn.textContent = '⏳ Downloading...';
    }

    addLog('🔧 Downloading Deno...');

    try {
        await App.DownloadDeno();
        addLog('✅ Deno installed successfully');
        alert('Deno installed successfully!');

        // Refresh the runtime status display
        await loadJSRuntimeStatus();
    } catch (err) {
        addLog('❌ Failed to download Deno: ' + err);
        alert('Failed to download Deno: ' + err);

        if (btn) {
            btn.disabled = false;
            btn.textContent = '⬇️ Download Bundled Deno';
        }
    }
};

// Log Tab
function renderLogTab() {
    const content = document.getElementById('tab-content');
    content.innerHTML = `
        <div class="h-full flex flex-col space-y-4">
            <div class="flex justify-between items-center">
                <h2 class="text-2xl font-semibold">Log Output</h2>
                <button onclick="window.clearLogs()" class="btn-secondary">
                    🗑️ Clear Logs
                </button>
            </div>

            <div class="flex-1 overflow-hidden">
                <div id="log-output" class="log-output h-full"></div>
            </div>
        </div>
    `;

    displayLogs();
}

function addLog(message, isVerbose = false) {
    if (isVerbose && !state.settings?.General?.VerboseLogging) {
        return;
    }
    const timestamp = new Date().toLocaleTimeString();
    state.logs.push(`[${timestamp}] ${message}`);

    if (state.logs.length > 1000) {
        state.logs = state.logs.slice(-1000);
    }

    if (state.currentTab === 'log') {
        displayLogs();
    }
}

function addVerboseLog(message) {
    addLog(message, true);
}

function displayLogs() {
    const output = document.getElementById('log-output');
    if (!output) return;

    if (state.logs.length === 0) {
        output.innerHTML = '<div class="text-gray-500">No logs yet...</div>';
        return;
    }

    // Only auto-scroll if user is already near the bottom
    const atBottom = output.scrollHeight - output.scrollTop - output.clientHeight < 40;
    output.innerHTML = state.logs.map(log => escapeHtml(log)).join('<br>');
    if (atBottom) {
        output.scrollTop = output.scrollHeight;
    }
}

window.clearLogs = function() {
    state.logs = [];
    displayLogs();
};

// Clipboard monitoring
let lastClipboard = '';
let clipboardMonitorEnabled = false;
let clipboardInterval = null;

async function startClipboardMonitoring() {
    // Wait for settings to load
    if (!state.settings) {
        await loadSettings();
    }

    if (!state.settings || !state.settings.General.ClipboardMonitoring) {
        clipboardMonitorEnabled = false;
        return;
    }

    clipboardMonitorEnabled = true;

    // Clear any existing interval
    if (clipboardInterval) {
        clearInterval(clipboardInterval);
    }

    clipboardInterval = setInterval(async () => {
        if (!clipboardMonitorEnabled) return;

        try {
            // Use backend to get clipboard (bypasses browser security restrictions)
            const text = await App.GetClipboard();
            if (text && text !== lastClipboard && text.length > 10) {
                lastClipboard = text;
                const url = extractURL(text);
                if (url && isVideoURL(url)) {
                    // Auto-paste to input if on download tab and input is empty
                    if (state.currentTab === 'download') {
                        const input = document.getElementById('url-input');
                        if (input && !input.value) {
                            input.value = url;
                            addLog(`📋 Auto-pasted URL from clipboard`);
                        }
                    }
                }
            }
        } catch (err) {
            // Clipboard access may have failed, silently ignore
        }
    }, 1000); // Poll every second
}

// URL pattern matching
const urlPatterns = [
    /https?:\/\/(?:www\.)?youtube\.com\/watch\?v=[\w-]+/,
    /https?:\/\/youtu\.be\/[\w-]+/,
    /https?:\/\/(?:www\.)?youtube\.com\/playlist\?list=[\w-]+/,
    /https?:\/\/(?:www\.)?youtube\.com\/shorts\/[\w-]+/,
    /https?:\/\/(?:www\.)?vimeo\.com\/\d+/,
    /https?:\/\/(?:www\.)?dailymotion\.com\/video\/[\w-]+/,
    /https?:\/\/(?:www\.)?twitch\.tv\/[\w-]+/,
    /https?:\/\/(?:www\.)?twitter\.com\/\w+\/status\/\d+/,
    /https?:\/\/(?:www\.)?x\.com\/\w+\/status\/\d+/,
    /https?:\/\/(?:www\.)?instagram\.com\/(?:p|reel)\/[\w-]+/,
    /https?:\/\/(?:www\.)?tiktok\.com\/@[\w.-]+\/video\/\d+/,
    /https?:\/\/(?:www\.)?reddit\.com\/r\/\w+\/comments\/[\w-]+/,
    /https?:\/\/(?:www\.)?soundcloud\.com\/[\w-]+\/[\w-]+/,
    /https?:\/\/(?:www\.)?facebook\.com\/[\w.]+\/videos\/\d+/,
    /https?:\/\/[^\s<>"]+/ // Generic fallback
];

function extractURL(text) {
    text = text.trim();
    for (const pattern of urlPatterns) {
        const match = text.match(pattern);
        if (match) {
            return match[0];
        }
    }
    return null;
}

function isVideoURL(url) {
    // Check if URL matches common video platforms
    const videoPatterns = [
        /youtube\.com/,
        /youtu\.be/,
        /vimeo\.com/,
        /dailymotion\.com/,
        /twitch\.tv/,
        /twitter\.com/,
        /x\.com/,
        /instagram\.com/,
        /tiktok\.com/,
        /reddit\.com/,
        /soundcloud\.com/,
        /facebook\.com/
    ];
    return videoPatterns.some(pattern => pattern.test(url));
}

// Setup event listeners from Go backend
function setupEventListeners() {
    // Listen for download updates
    EventsOn('download:update', (item) => {
        const prev = state.downloads[item.id];
        state.downloads[item.id] = item;
        if (state.currentTab === 'download') {
            updateDownloadQueue();
        }
        if (item.file_path && (!prev || prev.file_path !== item.file_path)) {
            addVerboseLog(`file_path set: "${item.file_path}"`);
        }
        addVerboseLog(`${item.title || item.url} - ${item.status}`);
    });

    // Listen for queue updates
    EventsOn('queue:update', (items) => {
        if (items && Array.isArray(items)) {
            // Treat as authoritative snapshot, not additive merge,
            // so removed items don't linger as ghost cards
            state.downloads = Object.fromEntries(items.map(item => [item.id, item]));
            if (state.currentTab === 'download') {
                updateDownloadQueue();
            }
        }
    });

    EventsOn('history:update', (entries) => {
        state.recentHistory = entries || [];
        if (state.currentTab === 'download') {
            updateDownloadQueue();
        }
    });

    // Listen for yt-dlp output logs
    EventsOn('download:log', (data) => {
        if (data && data.line) {
            const line = data.line;
            const isSignificant = !line.startsWith('[download]') ||
                line.includes('Destination:') ||
                line.includes('ERROR') ||
                line.includes('WARNING') ||
                line.includes('has already been downloaded') ||
                line.includes('Downloading item');
            if (isSignificant) {
                addLog(`📺 ${line}`);
            } else {
                addVerboseLog(`📺 ${line}`);
            }
        }
    });

    EventsOn('updater:progress', (message) => {
        addVerboseLog(`updater: ${message}`);
    });

    // Listen for errors
    EventsOn('error', (message) => {
        addLog(`❌ ${message}`);
        alert('Error: ' + message);
    });

    // Update status indicators — inline in the header bar
    EventsOn('update:checking', () => {
        showUpdateChecking();
    });

    EventsOn('update:available', (info) => {
        showUpdateAvailable(info);
    });

    EventsOn('update:none', () => {
        showNoUpdate();
    });

    // Listen for log messages
    EventsOn('log', (message) => {
        addLog(message);
    });

    // FFmpeg converter events
    EventsOn('convert:progress', (job) => {
        if (!job) return;
        state.conversion = job;
        syncConversionUI(job);

        if (job.status === 'completed') {
            state.conversion = null;
            addLog(`Conversion complete: ${job.output_file}`);
            showConvertResult('completed', job.output_file);
        } else if (job.status === 'failed') {
            state.conversion = null;
            addLog(`Conversion failed: ${job.error}`);
            showConvertResult('failed', job.error);
        }
    });

    EventsOn('convert:error', (message) => {
        addLog(`Conversion error: ${message}`);
        resetConvertUI();
    });

    EventsOn('convert:log', (line) => {
        addVerboseLog(`ffmpeg: ${line}`);
    });

    EventsOn('ffmpeg:progress', (message) => {
        addVerboseLog(`ffmpeg download: ${message}`);
        const el = document.getElementById('ffmpeg-progress');
        if (el) { el.textContent = message; el.classList.remove('hidden'); }
    });

    EventsOn('jsruntime:progress', (message) => {
        addVerboseLog(`jsruntime: ${message}`);
        const btn = document.getElementById('download-deno-btn');
        if (btn) {
            btn.textContent = `⏳ ${message}`;
        }
    });

    // Start clipboard monitoring
    startClipboardMonitoring();
}

// Update indicator — shown in the header bar
function showUpdateChecking() {
    const el = document.getElementById('update-indicator');
    if (!el) return;
    el.innerHTML = `
        <span class="animate-spin inline-block">♻️</span>
        <span class="text-gray-400">Checking for updates...</span>
    `;
}

function showUpdateAvailable(info) {
    const el = document.getElementById('update-indicator');
    if (!el) return;
    el.innerHTML = `
        <span class="text-yellow-400">♻️</span>
        <button id="update-now-btn" class="text-yellow-300 hover:text-yellow-100 underline cursor-pointer bg-transparent border-none text-sm">
            Update found, click here to update yt-dlp
        </button>
    `;
    document.getElementById('update-now-btn').addEventListener('click', async () => {
        el.innerHTML = `
            <span class="animate-spin inline-block">♻️</span>
            <span class="text-gray-400">Updating yt-dlp...</span>
        `;
        try {
            await App.UpdateYtDlp();
            addLog('✅ yt-dlp updated successfully');
            el.innerHTML = `
                <span class="text-green-400">✓</span>
                <span class="text-green-400">yt-dlp updated!</span>
            `;
            setTimeout(() => { el.innerHTML = ''; }, 5000);
        } catch (err) {
            addLog('❌ Update failed: ' + err);
            el.innerHTML = `
                <span class="text-red-400">✕</span>
                <span class="text-red-400">Update failed</span>
            `;
        }
    });
}

function showNoUpdate() {
    const el = document.getElementById('update-indicator');
    if (!el) return;
    el.innerHTML = `
        <span class="text-green-400">✓</span>
        <span class="text-green-400">No updates found</span>
    `;
    setTimeout(() => { el.innerHTML = ''; }, 4000);
}

// Manual check from settings tab — reuses the inline indicator
window.checkForUpdates = async function() {
    showUpdateChecking();
    try {
        const info = await App.CheckForUpdates();
        if (info && info.update_available) {
            showUpdateAvailable(info);
        } else {
            showNoUpdate();
        }
    } catch (err) {
        const el = document.getElementById('update-indicator');
        if (el) {
            el.innerHTML = `
                <span class="text-red-400">✕</span>
                <span class="text-red-400">Check failed</span>
            `;
        }
        addLog('❌ Failed to check for updates: ' + err);
    }
};

// Utility functions
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Escape for use inside JS string literals in onclick handlers
function escapeJsStr(str) {
    if (!str) return '';
    return str
        .replace(/\\/g, '\\\\')
        .replace(/'/g, "\\'")
        .replace(/\n/g, '\\n')
        .replace(/\r/g, '\\r')
        .replace(/\u2028/g, '\\u2028')
        .replace(/\u2029/g, '\\u2029');
}

// Custom context menu for text inputs
function setupContextMenu() {
    // Create context menu element
    const menu = document.createElement('div');
    menu.id = 'context-menu';
    menu.className = 'fixed bg-gray-800 border border-gray-600 rounded-lg shadow-lg py-1 z-50 hidden';
    menu.innerHTML = `
        <button id="ctx-cut" class="w-full px-4 py-2 text-left text-sm hover:bg-gray-700">Cut</button>
        <button id="ctx-copy" class="w-full px-4 py-2 text-left text-sm hover:bg-gray-700">Copy</button>
        <button id="ctx-paste" class="w-full px-4 py-2 text-left text-sm hover:bg-gray-700">Paste</button>
        <button id="ctx-selectall" class="w-full px-4 py-2 text-left text-sm hover:bg-gray-700">Select All</button>
    `;
    document.body.appendChild(menu);

    let targetElement = null;

    // Show menu on right-click for inputs/textareas
    document.addEventListener('contextmenu', async (e) => {
        const target = e.target;
        if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
            e.preventDefault();
            targetElement = target;

            menu.style.left = e.pageX + 'px';
            menu.style.top = e.pageY + 'px';
            menu.classList.remove('hidden');
        } else {
            menu.classList.add('hidden');
        }
    });

    // Hide menu on click elsewhere
    document.addEventListener('click', () => {
        menu.classList.add('hidden');
    });

    // Context menu actions
    document.getElementById('ctx-cut').addEventListener('click', () => {
        if (targetElement) {
            const start = targetElement.selectionStart;
            const end = targetElement.selectionEnd;
            const text = targetElement.value.substring(start, end);
            navigator.clipboard.writeText(text);
            targetElement.value = targetElement.value.substring(0, start) + targetElement.value.substring(end);
            targetElement.focus();
        }
    });

    document.getElementById('ctx-copy').addEventListener('click', () => {
        if (targetElement) {
            const start = targetElement.selectionStart;
            const end = targetElement.selectionEnd;
            const text = targetElement.value.substring(start, end);
            navigator.clipboard.writeText(text);
            targetElement.focus();
        }
    });

    document.getElementById('ctx-paste').addEventListener('click', async () => {
        if (targetElement) {
            try {
                // Try browser API first
                let text = '';
                try {
                    text = await navigator.clipboard.readText();
                } catch {
                    // Fall back to backend
                    text = await App.GetClipboard();
                }

                const start = targetElement.selectionStart;
                const end = targetElement.selectionEnd;
                targetElement.value = targetElement.value.substring(0, start) + text + targetElement.value.substring(end);
                targetElement.selectionStart = targetElement.selectionEnd = start + text.length;
                targetElement.focus();
            } catch (err) {
                console.error('Paste failed:', err);
            }
        }
    });

    document.getElementById('ctx-selectall').addEventListener('click', () => {
        if (targetElement) {
            targetElement.select();
            targetElement.focus();
        }
    });
}

// Re-check file existence when the app regains focus
document.addEventListener('visibilitychange', async () => {
    if (document.visibilityState === 'visible') {
        await Promise.all([loadRecentHistory(), loadQueueStatus()]);
        if (state.currentTab === 'download') {
            updateDownloadQueue();
        }
    }
});

// Initialize on load
document.addEventListener('DOMContentLoaded', () => {
    init();
    setupContextMenu();
});
