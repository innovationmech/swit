// Saga Dashboard JavaScript
(function() {
    'use strict';

    // Configuration
    const CONFIG = {
        API_BASE: '/api',
        REFRESH_INTERVAL: 5000, // 5 seconds
        SSE_RECONNECT_DELAY: 3000,
        PAGE_SIZE: 20
    };

    // State
    let state = {
        currentPage: 1,
        statusFilter: '',
        sagas: [],
        metrics: null,
        alerts: [],
        eventSource: null,
        connected: false,
        autoRefreshEnabled: true
    };

    // DOM Elements
    const elements = {
        // Metrics
        totalSagas: document.getElementById('totalSagas'),
        completedSagas: document.getElementById('completedSagas'),
        runningSagas: document.getElementById('runningSagas'),
        failedSagas: document.getElementById('failedSagas'),
        avgDuration: document.getElementById('avgDuration'),
        successRate: document.getElementById('successRate'),
        
        // Connection status
        connectionStatus: document.getElementById('connectionStatus'),
        lastUpdate: document.getElementById('lastUpdate'),
        
        // Alerts
        alertsSection: document.getElementById('alertsSection'),
        alertsList: document.getElementById('alertsList'),
        
        // Saga table
        sagaTableBody: document.getElementById('sagaTableBody'),
        statusFilter: document.getElementById('statusFilter'),
        refreshBtn: document.getElementById('refreshBtn'),
        
        // Pagination
        prevBtn: document.getElementById('prevBtn'),
        nextBtn: document.getElementById('nextBtn'),
        pageInfo: document.getElementById('pageInfo'),
        
        // Modals
        detailModal: document.getElementById('detailModal'),
        modalBody: document.getElementById('modalBody'),
        closeModal: document.getElementById('closeModal'),
        vizModal: document.getElementById('vizModal'),
        vizModalBody: document.getElementById('vizModalBody'),
        closeVizModal: document.getElementById('closeVizModal')
    };

    // API Functions
    const api = {
        async get(endpoint) {
            try {
                const response = await fetch(`${CONFIG.API_BASE}${endpoint}`);
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return await response.json();
            } catch (error) {
                console.error(`API Error (GET ${endpoint}):`, error);
                throw error;
            }
        },

        async post(endpoint, data = {}) {
            try {
                const response = await fetch(`${CONFIG.API_BASE}${endpoint}`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(data)
                });
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return await response.json();
            } catch (error) {
                console.error(`API Error (POST ${endpoint}):`, error);
                throw error;
            }
        },

        // Saga endpoints
        async getSagas(page = 1, status = '', pageSize = CONFIG.PAGE_SIZE) {
            let query = `?page=${page}&page_size=${pageSize}`;
            if (status) {
                query += `&status=${status}`;
            }
            return await this.get(`/sagas${query}`);
        },

        async getSagaDetails(sagaId) {
            return await this.get(`/sagas/${sagaId}`);
        },

        async cancelSaga(sagaId, reason = '') {
            return await this.post(`/sagas/${sagaId}/cancel`, { reason });
        },

        async retrySaga(sagaId) {
            return await this.post(`/sagas/${sagaId}/retry`);
        },

        // Metrics endpoints
        async getMetrics() {
            return await this.get('/metrics');
        },

        async getRealtimeMetrics() {
            return await this.get('/metrics/realtime');
        },

        // Visualization endpoints
        async getVisualization(sagaId) {
            return await this.get(`/sagas/${sagaId}/visualization`);
        },

        // Alerts endpoints
        async getAlerts() {
            return await this.get('/alerts');
        },

        async acknowledgeAlert(alertId) {
            return await this.post(`/alerts/${alertId}/acknowledge`);
        }
    };

    // UI Update Functions
    function updateConnectionStatus(connected) {
        state.connected = connected;
        elements.connectionStatus.className = `status-indicator ${connected ? 'connected' : 'disconnected'}`;
        elements.connectionStatus.innerHTML = `
            <span class="status-dot"></span>
            ${connected ? '已连接' : '未连接'}
        `;
    }

    function updateLastUpdateTime() {
        const now = new Date();
        elements.lastUpdate.textContent = now.toLocaleTimeString('zh-CN');
    }

    function updateMetrics(metrics) {
        if (!metrics) return;

        state.metrics = metrics;

        // Update metric cards
        elements.totalSagas.textContent = metrics.total_sagas || 0;
        elements.completedSagas.textContent = metrics.completed_sagas || 0;
        elements.runningSagas.textContent = metrics.running_sagas || 0;
        elements.failedSagas.textContent = metrics.failed_sagas || 0;

        // Update detailed metrics
        const avgDuration = metrics.avg_duration_ms || 0;
        elements.avgDuration.textContent = formatDuration(avgDuration);

        const successRate = metrics.total_sagas > 0 
            ? ((metrics.completed_sagas / metrics.total_sagas) * 100).toFixed(1)
            : 0;
        elements.successRate.textContent = `${successRate}%`;

        updateLastUpdateTime();
    }

    function updateAlerts(alerts) {
        if (!alerts || alerts.length === 0) {
            elements.alertsSection.style.display = 'none';
            return;
        }

        state.alerts = alerts;
        elements.alertsSection.style.display = 'block';
        
        elements.alertsList.innerHTML = alerts.map(alert => `
            <div class="alert-item">
                <div class="alert-info">
                    <div class="alert-title">${escapeHtml(alert.title || 'Alert')}</div>
                    <div class="alert-description">${escapeHtml(alert.description || '')}</div>
                    <div class="alert-meta" style="font-size: 0.85rem; color: #64748b; margin-top: 4px;">
                        ${alert.saga_id ? `Saga: ${alert.saga_id}` : ''} • 
                        ${alert.timestamp ? new Date(alert.timestamp).toLocaleString('zh-CN') : ''}
                    </div>
                </div>
                <div class="alert-actions">
                    ${alert.id ? `
                        <button class="btn btn-small btn-secondary" onclick="dashboard.acknowledgeAlert('${alert.id}')">
                            确认
                        </button>
                    ` : ''}
                </div>
            </div>
        `).join('');
    }

    function updateSagaTable(sagasData) {
        if (!sagasData || !sagasData.sagas) {
            elements.sagaTableBody.innerHTML = `
                <tr class="loading-row">
                    <td colspan="7">暂无数据</td>
                </tr>
            `;
            return;
        }

        state.sagas = sagasData.sagas;

        if (state.sagas.length === 0) {
            elements.sagaTableBody.innerHTML = `
                <tr class="loading-row">
                    <td colspan="7">暂无 Saga 数据</td>
                </tr>
            `;
            return;
        }

        elements.sagaTableBody.innerHTML = state.sagas.map(saga => {
            const progress = calculateProgress(saga);
            return `
                <tr>
                    <td>
                        <a href="#" onclick="dashboard.showSagaDetails('${saga.id}'); return false;" 
                           style="color: #3b82f6; text-decoration: none; font-family: monospace;">
                            ${saga.id.substring(0, 8)}...
                        </a>
                    </td>
                    <td>${escapeHtml(saga.type || 'N/A')}</td>
                    <td>
                        <span class="status-badge status-${saga.status}">
                            ${getStatusText(saga.status)}
                        </span>
                    </td>
                    <td>
                        <div class="progress-bar-container">
                            <div class="progress-bar" style="width: ${progress}%"></div>
                        </div>
                        <div style="font-size: 0.85rem; color: #64748b; margin-top: 4px;">
                            ${progress}%
                        </div>
                    </td>
                    <td style="font-size: 0.9rem;">
                        ${saga.started_at ? formatDateTime(saga.started_at) : 'N/A'}
                    </td>
                    <td>${formatDuration(saga.duration_ms || 0)}</td>
                    <td>
                        <div class="action-buttons">
                            <button class="btn btn-small btn-primary" 
                                    onclick="dashboard.showSagaDetails('${saga.id}')">
                                详情
                            </button>
                            <button class="btn btn-small btn-success" 
                                    onclick="dashboard.showVisualization('${saga.id}')">
                                流程图
                            </button>
                            ${saga.status === 'running' ? `
                                <button class="btn btn-small btn-danger" 
                                        onclick="dashboard.cancelSaga('${saga.id}')">
                                    取消
                                </button>
                            ` : ''}
                            ${saga.status === 'failed' || saga.status === 'compensated' ? `
                                <button class="btn btn-small btn-secondary" 
                                        onclick="dashboard.retrySaga('${saga.id}')">
                                    重试
                                </button>
                            ` : ''}
                        </div>
                    </td>
                </tr>
            `;
        }).join('');

        // Update pagination
        updatePagination(sagasData.pagination);
    }

    function updatePagination(pagination) {
        if (!pagination) return;

        const currentPage = pagination.current_page || state.currentPage;
        const totalPages = pagination.total_pages || 1;

        elements.pageInfo.textContent = `第 ${currentPage} / ${totalPages} 页`;
        elements.prevBtn.disabled = currentPage <= 1;
        elements.nextBtn.disabled = currentPage >= totalPages;
    }

    // Helper Functions
    function formatDuration(ms) {
        if (!ms || ms < 0) return '0 ms';
        if (ms < 1000) return `${Math.round(ms)} ms`;
        if (ms < 60000) return `${(ms / 1000).toFixed(1)} s`;
        return `${(ms / 60000).toFixed(1)} min`;
    }

    function formatDateTime(dateString) {
        if (!dateString) return 'N/A';
        try {
            return new Date(dateString).toLocaleString('zh-CN', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit'
            });
        } catch (e) {
            return dateString;
        }
    }

    function getStatusText(status) {
        const statusMap = {
            'running': '运行中',
            'completed': '已完成',
            'failed': '失败',
            'compensating': '补偿中',
            'compensated': '已补偿',
            'cancelled': '已取消'
        };
        return statusMap[status] || status;
    }

    function calculateProgress(saga) {
        if (!saga.steps || saga.steps.length === 0) return 0;
        const completed = saga.steps.filter(s => s.status === 'completed').length;
        return Math.round((completed / saga.steps.length) * 100);
    }

    function escapeHtml(text) {
        const map = {
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            '"': '&quot;',
            "'": '&#039;'
        };
        return String(text || '').replace(/[&<>"']/g, m => map[m]);
    }

    // Data Loading Functions
    async function loadMetrics() {
        try {
            const metrics = await api.getRealtimeMetrics();
            updateMetrics(metrics);
            updateConnectionStatus(true);
        } catch (error) {
            console.error('Failed to load metrics:', error);
            updateConnectionStatus(false);
        }
    }

    async function loadAlerts() {
        try {
            const response = await api.getAlerts();
            updateAlerts(response.alerts || []);
        } catch (error) {
            console.error('Failed to load alerts:', error);
        }
    }

    async function loadSagas() {
        try {
            const data = await api.getSagas(state.currentPage, state.statusFilter);
            updateSagaTable(data);
            updateConnectionStatus(true);
        } catch (error) {
            console.error('Failed to load sagas:', error);
            updateConnectionStatus(false);
            elements.sagaTableBody.innerHTML = `
                <tr class="loading-row">
                    <td colspan="7" style="color: #ef4444;">加载失败: ${escapeHtml(error.message)}</td>
                </tr>
            `;
        }
    }

    async function refreshAll() {
        await Promise.all([
            loadMetrics(),
            loadAlerts(),
            loadSagas()
        ]);
    }

    // Saga Detail Modal
    async function showSagaDetails(sagaId) {
        try {
            const saga = await api.getSagaDetails(sagaId);
            
            const stepsHtml = saga.steps && saga.steps.length > 0 ? `
                <div class="detail-section">
                    <h3>执行步骤</h3>
                    <div class="steps-list">
                        ${saga.steps.map((step, index) => `
                            <div class="step-item ${step.status}">
                                <div class="step-header">
                                    <span class="step-name">${index + 1}. ${escapeHtml(step.name || 'Step')}</span>
                                    <span class="status-badge status-${step.status}">
                                        ${getStatusText(step.status)}
                                    </span>
                                </div>
                                <div class="step-meta">
                                    ${step.started_at ? `<span>开始: ${formatDateTime(step.started_at)}</span>` : ''}
                                    ${step.completed_at ? `<span>完成: ${formatDateTime(step.completed_at)}</span>` : ''}
                                    ${step.duration_ms ? `<span>耗时: ${formatDuration(step.duration_ms)}</span>` : ''}
                                </div>
                                ${step.error ? `
                                    <div style="margin-top: 8px; padding: 8px; background: #fee2e2; border-radius: 4px; color: #991b1b; font-size: 0.85rem;">
                                        错误: ${escapeHtml(step.error)}
                                    </div>
                                ` : ''}
                            </div>
                        `).join('')}
                    </div>
                </div>
            ` : '';

            elements.modalBody.innerHTML = `
                <div class="detail-section">
                    <h3>基本信息</h3>
                    <div class="detail-grid">
                        <div class="detail-item">
                            <div class="detail-label">Saga ID</div>
                            <div class="detail-value" style="font-family: monospace; font-size: 0.9rem;">
                                ${saga.id}
                            </div>
                        </div>
                        <div class="detail-item">
                            <div class="detail-label">类型</div>
                            <div class="detail-value">${escapeHtml(saga.type || 'N/A')}</div>
                        </div>
                        <div class="detail-item">
                            <div class="detail-label">状态</div>
                            <div class="detail-value">
                                <span class="status-badge status-${saga.status}">
                                    ${getStatusText(saga.status)}
                                </span>
                            </div>
                        </div>
                        <div class="detail-item">
                            <div class="detail-label">开始时间</div>
                            <div class="detail-value">${formatDateTime(saga.started_at)}</div>
                        </div>
                        ${saga.completed_at ? `
                            <div class="detail-item">
                                <div class="detail-label">完成时间</div>
                                <div class="detail-value">${formatDateTime(saga.completed_at)}</div>
                            </div>
                        ` : ''}
                        <div class="detail-item">
                            <div class="detail-label">执行时间</div>
                            <div class="detail-value">${formatDuration(saga.duration_ms || 0)}</div>
                        </div>
                    </div>
                </div>
                
                ${saga.error ? `
                    <div class="detail-section">
                        <h3>错误信息</h3>
                        <div style="padding: 12px; background: #fee2e2; border-radius: 8px; color: #991b1b;">
                            ${escapeHtml(saga.error)}
                        </div>
                    </div>
                ` : ''}

                ${stepsHtml}
            `;

            elements.detailModal.classList.add('active');
        } catch (error) {
            console.error('Failed to load saga details:', error);
            alert('加载 Saga 详情失败: ' + error.message);
        }
    }

    // Visualization Modal
    async function showVisualization(sagaId) {
        try {
            const viz = await api.getVisualization(sagaId);
            
            if (!viz.steps || viz.steps.length === 0) {
                elements.vizModalBody.innerHTML = `
                    <div class="viz-container">
                        <p style="text-align: center; color: #64748b;">暂无流程可视化数据</p>
                    </div>
                `;
            } else {
                elements.vizModalBody.innerHTML = `
                    <div class="viz-container">
                        <div class="flow-diagram">
                            ${viz.steps.map(step => `
                                <div class="flow-step ${step.status}">
                                    <div style="font-weight: 600; margin-bottom: 8px;">
                                        ${escapeHtml(step.name || 'Step')}
                                    </div>
                                    <div style="font-size: 0.9rem; color: #64748b;">
                                        状态: ${getStatusText(step.status)}
                                    </div>
                                    ${step.duration_ms ? `
                                        <div style="font-size: 0.85rem; color: #64748b; margin-top: 4px;">
                                            耗时: ${formatDuration(step.duration_ms)}
                                        </div>
                                    ` : ''}
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            }

            elements.vizModal.classList.add('active');
        } catch (error) {
            console.error('Failed to load visualization:', error);
            alert('加载流程可视化失败: ' + error.message);
        }
    }

    // Action Functions
    async function cancelSaga(sagaId) {
        if (!confirm('确定要取消这个 Saga 吗？')) return;

        try {
            await api.cancelSaga(sagaId, '用户手动取消');
            alert('Saga 已取消');
            await refreshAll();
        } catch (error) {
            console.error('Failed to cancel saga:', error);
            alert('取消 Saga 失败: ' + error.message);
        }
    }

    async function retrySaga(sagaId) {
        if (!confirm('确定要重试这个 Saga 吗？')) return;

        try {
            await api.retrySaga(sagaId);
            alert('Saga 已提交重试');
            await refreshAll();
        } catch (error) {
            console.error('Failed to retry saga:', error);
            alert('重试 Saga 失败: ' + error.message);
        }
    }

    async function acknowledgeAlert(alertId) {
        try {
            await api.acknowledgeAlert(alertId);
            await loadAlerts();
        } catch (error) {
            console.error('Failed to acknowledge alert:', error);
            alert('确认告警失败: ' + error.message);
        }
    }

    // Event Handlers
    function setupEventListeners() {
        // Refresh button
        elements.refreshBtn.addEventListener('click', refreshAll);

        // Status filter
        elements.statusFilter.addEventListener('change', (e) => {
            state.statusFilter = e.target.value;
            state.currentPage = 1;
            loadSagas();
        });

        // Pagination
        elements.prevBtn.addEventListener('click', () => {
            if (state.currentPage > 1) {
                state.currentPage--;
                loadSagas();
            }
        });

        elements.nextBtn.addEventListener('click', () => {
            state.currentPage++;
            loadSagas();
        });

        // Modal close buttons
        elements.closeModal.addEventListener('click', () => {
            elements.detailModal.classList.remove('active');
        });

        elements.closeVizModal.addEventListener('click', () => {
            elements.vizModal.classList.remove('active');
        });

        // Close modals on background click
        elements.detailModal.addEventListener('click', (e) => {
            if (e.target === elements.detailModal) {
                elements.detailModal.classList.remove('active');
            }
        });

        elements.vizModal.addEventListener('click', (e) => {
            if (e.target === elements.vizModal) {
                elements.vizModal.classList.remove('active');
            }
        });

        // Close modals on Escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                elements.detailModal.classList.remove('active');
                elements.vizModal.classList.remove('active');
            }
        });
    }

    // Auto-refresh
    function startAutoRefresh() {
        if (state.autoRefreshEnabled) {
            setInterval(refreshAll, CONFIG.REFRESH_INTERVAL);
        }
    }

    // Initialize
    async function init() {
        console.log('Initializing Saga Dashboard...');
        
        setupEventListeners();
        
        // Initial data load
        await refreshAll();
        
        // Start auto-refresh
        startAutoRefresh();
        
        console.log('Saga Dashboard initialized successfully');
    }

    // Expose public API
    window.dashboard = {
        showSagaDetails,
        showVisualization,
        cancelSaga,
        retrySaga,
        acknowledgeAlert,
        refresh: refreshAll
    };

    // Start when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();

