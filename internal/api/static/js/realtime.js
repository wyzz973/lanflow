(function() {
    var ws = null;
    var lastData = null;
    var lastTime = null;

    var container = document.getElementById('realtime');
    container.innerHTML =
        '<div class="summary-cards">' +
            '<div class="summary-card">' +
                '<div class="icon blue">' +
                    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>' +
                '</div>' +
                '<div class="label">在线设备</div>' +
                '<div class="value" id="rt-device-count">0</div>' +
            '</div>' +
            '<div class="summary-card">' +
                '<div class="icon green">' +
                    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/></svg>' +
                '</div>' +
                '<div class="label">总上行</div>' +
                '<div class="value" id="rt-total-tx">0 B</div>' +
            '</div>' +
            '<div class="summary-card">' +
                '<div class="icon purple">' +
                    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/></svg>' +
                '</div>' +
                '<div class="label">总下行</div>' +
                '<div class="value" id="rt-total-rx">0 B</div>' +
            '</div>' +
        '</div>' +
        '<div class="card">' +
            '<div class="card-header">' +
                '<h3>' +
                    '<span class="status-dot active" id="ws-status"></span>' +
                    '实时流量监控' +
                '</h3>' +
            '</div>' +
            '<div class="table-wrapper">' +
                '<table class="data-table">' +
                    '<thead><tr>' +
                        '<th>设备名称</th>' +
                        '<th>IP 地址</th>' +
                        '<th>上行速率</th>' +
                        '<th>下行速率</th>' +
                        '<th>今日上行</th>' +
                        '<th>今日下行</th>' +
                        '<th>今日总计</th>' +
                        '<th>操作</th>' +
                    '</tr></thead>' +
                    '<tbody id="realtime-body"></tbody>' +
                '</table>' +
            '</div>' +
        '</div>';

    function connect() {
        var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(proto + '//' + location.host + '/ws/realtime');

        ws.onopen = function() {
            var dot = document.getElementById('ws-status');
            if (dot) dot.className = 'status-dot active';
        };

        ws.onclose = function() {
            var dot = document.getElementById('ws-status');
            if (dot) dot.className = 'status-dot inactive';
            setTimeout(connect, 3000);
        };

        ws.onerror = function() {
            var dot = document.getElementById('ws-status');
            if (dot) dot.className = 'status-dot inactive';
        };

        ws.onmessage = function(event) {
            var msg = JSON.parse(event.data);
            var now = Date.now();
            var currentData = {};

            (msg.data || []).forEach(function(d) { currentData[d.ip] = d; });

            if (lastData && lastTime) {
                var elapsed = (now - lastTime) / 1000;
                if (elapsed > 0) renderTable(currentData, lastData, elapsed);
            } else {
                renderTable(currentData, {}, 1);
            }

            lastData = currentData;
            lastTime = now;
        };
    }

    function renderTable(current, previous, elapsed) {
        var tbody = document.getElementById('realtime-body');
        var rows = [];
        var totalTx = 0;
        var totalRx = 0;

        for (var ip in current) {
            var cur = current[ip];
            var prev = previous[ip] || { tx_bytes: cur.tx_bytes, rx_bytes: cur.rx_bytes };
            var txRate = Math.max(0, (cur.tx_bytes - prev.tx_bytes) / elapsed);
            var rxRate = Math.max(0, (cur.rx_bytes - prev.rx_bytes) / elapsed);

            totalTx += cur.tx_bytes;
            totalRx += cur.rx_bytes;

            rows.push({
                name: cur.name || ip,
                ip: ip,
                txRate: txRate,
                rxRate: rxRate,
                txTotal: cur.tx_bytes,
                rxTotal: cur.rx_bytes,
                total: cur.tx_bytes + cur.rx_bytes
            });
        }

        rows.sort(function(a, b) { return b.total - a.total; });

        // Update summary cards
        var countEl = document.getElementById('rt-device-count');
        var txEl = document.getElementById('rt-total-tx');
        var rxEl = document.getElementById('rt-total-rx');
        if (countEl) countEl.textContent = rows.length;
        if (txEl) txEl.textContent = Utils.formatBytes(totalTx);
        if (rxEl) rxEl.textContent = Utils.formatBytes(totalRx);

        tbody.innerHTML = rows.map(function(r) {
            return '<tr>' +
                '<td><strong>' + Utils.escapeHtml(r.name) + '</strong></td>' +
                '<td class="mono">' + Utils.escapeHtml(r.ip) + '</td>' +
                '<td>' + Utils.formatRate(r.txRate) + '</td>' +
                '<td>' + Utils.formatRate(r.rxRate) + '</td>' +
                '<td>' + Utils.formatBytes(r.txTotal) + '</td>' +
                '<td>' + Utils.formatBytes(r.rxTotal) + '</td>' +
                '<td><strong>' + Utils.formatBytes(r.total) + '</strong></td>' +
                '<td><button class="btn btn-sm btn-outline" onclick="Realtime.showDomains(\'' + Utils.escapeHtml(r.ip) + '\',\'' + Utils.escapeHtml(r.name) + '\')">详情</button></td>' +
            '</tr>';
        }).join('');
    }

    function showDomains(ip, name) {
        var date = Utils.todayStr();

        // Create modal
        var overlay = document.createElement('div');
        overlay.className = 'modal-overlay';
        overlay.innerHTML =
            '<div class="modal">' +
                '<div class="modal-header">' +
                    '<h3>流量明细 — ' + Utils.escapeHtml(name) + '</h3>' +
                    '<button class="modal-close" id="modal-close-btn">&times;</button>' +
                '</div>' +
                '<div id="modal-domain-content">' +
                    '<p style="color:#9ca3af;text-align:center;padding:20px;">加载中...</p>' +
                '</div>' +
            '</div>';

        document.body.appendChild(overlay);

        // Close handlers
        var closeBtn = document.getElementById('modal-close-btn');
        closeBtn.addEventListener('click', function() { document.body.removeChild(overlay); });
        overlay.addEventListener('click', function(e) {
            if (e.target === overlay) document.body.removeChild(overlay);
        });

        // Fetch domain data
        fetch('/api/domains/' + encodeURIComponent(ip) + '?range=day&date=' + date)
            .then(function(r) { return r.json(); })
            .then(function(resp) {
                var data = resp.data || [];
                renderDomainModal(data);
            })
            .catch(function(err) {
                var contentEl = document.getElementById('modal-domain-content');
                if (contentEl) contentEl.innerHTML = '<p style="color:#ef4444;text-align:center;padding:20px;">加载失败</p>';
            });
    }

    function renderDomainModal(data) {
        var contentEl = document.getElementById('modal-domain-content');
        if (!contentEl) return;

        if (!data || data.length === 0) {
            contentEl.innerHTML = '<div class="empty-state"><p>暂无域名流量数据</p></div>';
            return;
        }

        // Sort by total descending
        data.sort(function(a, b) { return (b.tx_bytes + b.rx_bytes) - (a.tx_bytes + a.rx_bytes); });

        var grandTotal = data.reduce(function(sum, d) { return sum + d.tx_bytes + d.rx_bytes; }, 0);

        var html = '<div class="table-wrapper"><table class="data-table">' +
            '<thead><tr>' +
                '<th>域名</th><th>上行</th><th>下行</th><th>总计</th><th>占比</th>' +
            '</tr></thead><tbody>';

        data.forEach(function(d) {
            var total = d.tx_bytes + d.rx_bytes;
            var pct = grandTotal > 0 ? (total / grandTotal * 100) : 0;
            html += '<tr>' +
                '<td class="mono">' + Utils.escapeHtml(d.domain || d.name || '-') + '</td>' +
                '<td>' + Utils.formatBytes(d.tx_bytes) + '</td>' +
                '<td>' + Utils.formatBytes(d.rx_bytes) + '</td>' +
                '<td><strong>' + Utils.formatBytes(total) + '</strong></td>' +
                '<td>' +
                    '<div style="display:flex;align-items:center;gap:8px;">' +
                        '<div class="progress-bar"><div class="progress-bar-fill" style="width:' + pct.toFixed(1) + '%"></div></div>' +
                        '<span style="font-size:13px;color:#6b7280;min-width:48px;">' + pct.toFixed(1) + '%</span>' +
                    '</div>' +
                '</td>' +
            '</tr>';
        });

        html += '</tbody></table></div>';
        contentEl.innerHTML = html;
    }

    window.Realtime = {
        activate: function() {
            if (!ws || ws.readyState === WebSocket.CLOSED) connect();
        },
        showDomains: showDomains
    };

    connect();
})();
