(function() {
    let ws = null;
    let lastData = null;
    let lastTime = null;

    const container = document.getElementById('realtime');
    container.innerHTML = `
        <div class="card">
            <h3><span class="status-dot active" id="ws-status"></span>实时流量监控</h3>
            <table class="data-table">
                <thead>
                    <tr>
                        <th>设备名称</th>
                        <th>IP 地址</th>
                        <th>上行速率</th>
                        <th>下行速率</th>
                        <th>今日上行</th>
                        <th>今日下行</th>
                        <th>今日总计</th>
                    </tr>
                </thead>
                <tbody id="realtime-body"></tbody>
            </table>
        </div>
    `;

    function connect() {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(proto + '//' + location.host + '/ws/realtime');

        ws.onopen = function() {
            document.getElementById('ws-status').className = 'status-dot active';
        };

        ws.onclose = function() {
            document.getElementById('ws-status').className = 'status-dot inactive';
            setTimeout(connect, 3000);
        };

        ws.onmessage = function(event) {
            const msg = JSON.parse(event.data);
            const now = Date.now();
            const currentData = {};

            (msg.data || []).forEach(d => { currentData[d.ip] = d; });

            if (lastData && lastTime) {
                const elapsed = (now - lastTime) / 1000;
                renderTable(currentData, lastData, elapsed);
            }

            lastData = currentData;
            lastTime = now;
        };
    }

    function renderTable(current, previous, elapsed) {
        const tbody = document.getElementById('realtime-body');
        const rows = [];

        for (const ip in current) {
            const cur = current[ip];
            const prev = previous[ip] || { tx_bytes: 0, rx_bytes: 0 };
            const txRate = Math.max(0, (cur.tx_bytes - prev.tx_bytes) / elapsed);
            const rxRate = Math.max(0, (cur.rx_bytes - prev.rx_bytes) / elapsed);

            rows.push({
                name: cur.name,
                ip: ip,
                txRate: txRate,
                rxRate: rxRate,
                txTotal: cur.tx_bytes,
                rxTotal: cur.rx_bytes,
                total: cur.tx_bytes + cur.rx_bytes
            });
        }

        rows.sort((a, b) => b.total - a.total);

        tbody.innerHTML = rows.map(r => `
            <tr>
                <td>${r.name}</td>
                <td>${r.ip}</td>
                <td>${Utils.formatRate(r.txRate)}</td>
                <td>${Utils.formatRate(r.rxRate)}</td>
                <td>${Utils.formatBytes(r.txTotal)}</td>
                <td>${Utils.formatBytes(r.rxTotal)}</td>
                <td>${Utils.formatBytes(r.total)}</td>
            </tr>
        `).join('');
    }

    window.Realtime = {
        activate: function() {
            if (!ws || ws.readyState === WebSocket.CLOSED) connect();
        }
    };

    connect();
})();
