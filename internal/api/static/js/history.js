(function() {
    var chart = null;
    var detailChart = null;
    var lastQueryData = [];
    var selectedIp = null;

    var container = document.getElementById('history');
    container.innerHTML =
        '<div class="controls">' +
            '<select id="hist-range">' +
                '<option value="day">按天</option>' +
                '<option value="week">按周</option>' +
                '<option value="month">按月</option>' +
            '</select>' +
            '<input type="date" id="hist-date" value="' + Utils.todayStr() + '">' +
            '<button class="btn" id="hist-query">' +
                '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:16px;height:16px;"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>' +
                '查询' +
            '</button>' +
        '</div>' +
        '<div class="card">' +
            '<div class="card-header"><h3>流量对比</h3></div>' +
            '<div class="chart-container" id="hist-chart"></div>' +
        '</div>' +
        '<div class="card" id="hist-summary-card">' +
            '<div class="card-header"><h3>流量排行</h3></div>' +
            '<div class="table-wrapper">' +
                '<table class="data-table">' +
                    '<thead><tr>' +
                        '<th>排名</th><th>设备名称</th><th>IP 地址</th><th>上行</th><th>下行</th><th>总计</th>' +
                    '</tr></thead>' +
                    '<tbody id="hist-summary-body"></tbody>' +
                '</table>' +
            '</div>' +
        '</div>' +
        '<div class="card" id="detail-card" style="display:none;">' +
            '<div class="card-header"><h3>详细趋势 — <span id="detail-title"></span></h3></div>' +
            '<div class="chart-container" id="detail-chart"></div>' +
        '</div>' +
        '<div class="card domain-section" id="hist-domain-card" style="display:none;">' +
            '<div class="card-header"><h3>域名流量 — <span id="hist-domain-title"></span></h3></div>' +
            '<div class="table-wrapper">' +
                '<table class="data-table">' +
                    '<thead><tr>' +
                        '<th>域名</th><th>上行</th><th>下行</th><th>总计</th><th>占比</th>' +
                    '</tr></thead>' +
                    '<tbody id="hist-domain-body"></tbody>' +
                '</table>' +
            '</div>' +
        '</div>';

    document.getElementById('hist-query').addEventListener('click', query);

    function query() {
        var range = document.getElementById('hist-range').value;
        var date = document.getElementById('hist-date').value;

        fetch('/api/stats?range=' + range + '&date=' + date)
            .then(function(r) { return r.json(); })
            .then(function(resp) {
                lastQueryData = resp.data || [];
                renderBarChart(lastQueryData);
                renderSummaryTable(lastQueryData);
                // Hide detail panels
                document.getElementById('detail-card').style.display = 'none';
                document.getElementById('hist-domain-card').style.display = 'none';
                selectedIp = null;
            })
            .catch(function(err) { console.error('query stats:', err); });
    }

    function renderBarChart(data) {
        if (!chart) {
            chart = echarts.init(document.getElementById('hist-chart'));
            window.addEventListener('resize', function() { chart && chart.resize(); });
        }

        data.sort(function(a, b) { return (b.tx_bytes + b.rx_bytes) - (a.tx_bytes + a.rx_bytes); });

        var names = data.map(function(d) { return d.name || d.ip; });
        var tx = data.map(function(d) { return d.tx_bytes; });
        var rx = data.map(function(d) { return d.rx_bytes; });
        var ips = data.map(function(d) { return d.ip; });

        chart.setOption({
            tooltip: {
                trigger: 'axis',
                formatter: function(params) {
                    var s = Utils.escapeHtml(params[0].name) + '<br/>';
                    params.forEach(function(p) {
                        s += p.marker + ' ' + p.seriesName + ': ' + Utils.formatBytes(p.value) + '<br/>';
                    });
                    return s;
                }
            },
            legend: { data: ['上行', '下行'], bottom: 0 },
            grid: { top: 20, right: 20, bottom: 40, left: 80 },
            xAxis: {
                type: 'category',
                data: names,
                axisLabel: { rotate: names.length > 6 ? 30 : 0, fontSize: 12 }
            },
            yAxis: {
                type: 'value',
                axisLabel: { formatter: function(v) { return Utils.formatBytes(v); } }
            },
            series: [
                {
                    name: '上行',
                    type: 'bar',
                    data: tx,
                    itemStyle: { color: '#3b82f6', borderRadius: [4, 4, 0, 0] },
                    barMaxWidth: 40
                },
                {
                    name: '下行',
                    type: 'bar',
                    data: rx,
                    itemStyle: { color: '#22c55e', borderRadius: [4, 4, 0, 0] },
                    barMaxWidth: 40
                }
            ]
        }, true);

        chart.off('click');
        chart.on('click', function(params) {
            var ip = ips[params.dataIndex];
            var name = data[params.dataIndex].name || ip;
            selectDevice(ip, name);
        });
    }

    function renderSummaryTable(data) {
        var sorted = data.slice().sort(function(a, b) {
            return (b.tx_bytes + b.rx_bytes) - (a.tx_bytes + a.rx_bytes);
        });

        var tbody = document.getElementById('hist-summary-body');
        if (sorted.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="empty-state"><p>暂无数据</p></td></tr>';
            return;
        }

        tbody.innerHTML = sorted.map(function(d, i) {
            var rank = i + 1;
            var total = d.tx_bytes + d.rx_bytes;
            var rankClass = '';
            var rowClass = '';
            if (rank === 1) { rankClass = 'rank-gold'; rowClass = 'rank-row-gold'; }
            else if (rank === 2) { rankClass = 'rank-silver'; rowClass = 'rank-row-silver'; }
            else if (rank === 3) { rankClass = 'rank-bronze'; rowClass = 'rank-row-bronze'; }

            var selectedClass = (d.ip === selectedIp) ? ' selected' : '';

            return '<tr class="clickable-row ' + rowClass + selectedClass + '" data-ip="' + Utils.escapeHtml(d.ip) + '" data-name="' + Utils.escapeHtml(d.name || d.ip) + '">' +
                '<td class="rank-cell ' + rankClass + '">' + rank + '</td>' +
                '<td><strong>' + Utils.escapeHtml(d.name || d.ip) + '</strong></td>' +
                '<td class="mono">' + Utils.escapeHtml(d.ip) + '</td>' +
                '<td>' + Utils.formatBytes(d.tx_bytes) + '</td>' +
                '<td>' + Utils.formatBytes(d.rx_bytes) + '</td>' +
                '<td><strong>' + Utils.formatBytes(total) + '</strong></td>' +
            '</tr>';
        }).join('');

        // Add click handlers to rows
        var tableRows = tbody.querySelectorAll('.clickable-row');
        tableRows.forEach(function(row) {
            row.addEventListener('click', function() {
                var ip = row.dataset.ip;
                var name = row.dataset.name;
                selectDevice(ip, name);
            });
        });
    }

    function selectDevice(ip, name) {
        selectedIp = ip;

        // Highlight selected row
        var allRows = document.querySelectorAll('#hist-summary-body .clickable-row');
        allRows.forEach(function(r) { r.classList.remove('selected'); });
        var selectedRow = document.querySelector('#hist-summary-body .clickable-row[data-ip="' + ip + '"]');
        if (selectedRow) selectedRow.classList.add('selected');

        showDetail(ip, name);
        showDomainBreakdown(ip, name);
    }

    function showDetail(ip, name) {
        var range = document.getElementById('hist-range').value;
        var date = document.getElementById('hist-date').value;

        fetch('/api/stats/' + encodeURIComponent(ip) + '?range=' + range + '&date=' + date)
            .then(function(r) { return r.json(); })
            .then(function(resp) { renderDetailChart(resp.data || [], name); })
            .catch(function(err) { console.error('query detail:', err); });
    }

    function showDomainBreakdown(ip, name) {
        var range = document.getElementById('hist-range').value;
        var date = document.getElementById('hist-date').value;

        document.getElementById('hist-domain-card').style.display = 'block';
        document.getElementById('hist-domain-title').textContent = name;

        var tbody = document.getElementById('hist-domain-body');
        tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#9ca3af;padding:20px;">加载中...</td></tr>';

        fetch('/api/domains/' + encodeURIComponent(ip) + '?range=' + range + '&date=' + date)
            .then(function(r) { return r.json(); })
            .then(function(resp) {
                var data = resp.data || [];
                renderDomainTable(data);
            })
            .catch(function(err) {
                tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:#ef4444;padding:20px;">加载失败</td></tr>';
            });
    }

    function renderDomainTable(data) {
        var tbody = document.getElementById('hist-domain-body');

        if (!data || data.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="empty-state"><p>暂无域名流量数据</p></td></tr>';
            return;
        }

        data.sort(function(a, b) { return (b.tx_bytes + b.rx_bytes) - (a.tx_bytes + a.rx_bytes); });
        var grandTotal = data.reduce(function(sum, d) { return sum + d.tx_bytes + d.rx_bytes; }, 0);

        tbody.innerHTML = data.map(function(d) {
            var total = d.tx_bytes + d.rx_bytes;
            var pct = grandTotal > 0 ? (total / grandTotal * 100) : 0;
            return '<tr>' +
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
        }).join('');
    }

    function renderDetailChart(data, name) {
        document.getElementById('detail-card').style.display = 'block';
        document.getElementById('detail-title').textContent = name;

        if (!detailChart) {
            detailChart = echarts.init(document.getElementById('detail-chart'));
            window.addEventListener('resize', function() { detailChart && detailChart.resize(); });
        }

        var times = data.map(function(d) {
            var dt = new Date(d.timestamp * 1000);
            return dt.getHours().toString().padStart(2, '0') + ':' + dt.getMinutes().toString().padStart(2, '0');
        });
        var tx = data.map(function(d) { return d.tx_bytes; });
        var rx = data.map(function(d) { return d.rx_bytes; });

        detailChart.setOption({
            tooltip: {
                trigger: 'axis',
                formatter: function(params) {
                    var s = params[0].name + '<br/>';
                    params.forEach(function(p) {
                        s += p.marker + ' ' + p.seriesName + ': ' + Utils.formatBytes(p.value) + '<br/>';
                    });
                    return s;
                }
            },
            legend: { data: ['上行', '下行'], bottom: 0 },
            grid: { top: 20, right: 20, bottom: 40, left: 80 },
            xAxis: { type: 'category', data: times },
            yAxis: {
                type: 'value',
                axisLabel: { formatter: function(v) { return Utils.formatBytes(v); } }
            },
            series: [
                {
                    name: '上行',
                    type: 'line',
                    data: tx,
                    smooth: true,
                    showSymbol: false,
                    lineStyle: { width: 2 },
                    areaStyle: { color: 'rgba(59,130,246,0.1)' },
                    itemStyle: { color: '#3b82f6' }
                },
                {
                    name: '下行',
                    type: 'line',
                    data: rx,
                    smooth: true,
                    showSymbol: false,
                    lineStyle: { width: 2 },
                    areaStyle: { color: 'rgba(34,197,94,0.1)' },
                    itemStyle: { color: '#22c55e' }
                }
            ]
        }, true);
    }

    window.History = {
        activate: function() {
            if (!chart) query();
            else {
                setTimeout(function() {
                    chart && chart.resize();
                    detailChart && detailChart.resize();
                }, 100);
            }
        }
    };
})();
