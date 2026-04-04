(function() {
    let chart = null;
    let detailChart = null;

    const container = document.getElementById('history');
    container.innerHTML = `
        <div class="controls">
            <select id="hist-range">
                <option value="day">按天</option>
                <option value="week">按周</option>
                <option value="month">按月</option>
            </select>
            <input type="date" id="hist-date" value="${new Date().toISOString().split('T')[0]}">
            <button id="hist-query">查询</button>
        </div>
        <div class="card">
            <h3>流量对比</h3>
            <div class="chart-container" id="hist-chart"></div>
        </div>
        <div class="card" id="detail-card" style="display:none;">
            <h3>详细趋势 — <span id="detail-title"></span></h3>
            <div class="chart-container" id="detail-chart"></div>
        </div>
    `;

    document.getElementById('hist-query').addEventListener('click', query);

    function query() {
        const range = document.getElementById('hist-range').value;
        const date = document.getElementById('hist-date').value;

        fetch('/api/stats?range=' + range + '&date=' + date)
            .then(r => r.json())
            .then(resp => renderBarChart(resp.data || []))
            .catch(err => console.error('query stats:', err));
    }

    function renderBarChart(data) {
        if (!chart) {
            chart = echarts.init(document.getElementById('hist-chart'));
        }

        const names = data.map(d => d.name || d.ip);
        const tx = data.map(d => d.tx_bytes);
        const rx = data.map(d => d.rx_bytes);
        const ips = data.map(d => d.ip);

        chart.setOption({
            tooltip: {
                trigger: 'axis',
                formatter: function(params) {
                    let s = params[0].name + '<br/>';
                    params.forEach(p => {
                        s += p.marker + p.seriesName + ': ' + Utils.formatBytes(p.value) + '<br/>';
                    });
                    return s;
                }
            },
            legend: { data: ['上行', '下行'] },
            xAxis: { type: 'category', data: names, axisLabel: { rotate: 30 } },
            yAxis: {
                type: 'value',
                axisLabel: { formatter: function(v) { return Utils.formatBytes(v); } }
            },
            series: [
                { name: '上行', type: 'bar', data: tx, itemStyle: { color: '#1a73e8' } },
                { name: '下行', type: 'bar', data: rx, itemStyle: { color: '#34a853' } }
            ]
        });

        chart.off('click');
        chart.on('click', function(params) {
            const ip = ips[params.dataIndex];
            showDetail(ip, data[params.dataIndex].name || ip);
        });
    }

    function showDetail(ip, name) {
        const range = document.getElementById('hist-range').value;
        const date = document.getElementById('hist-date').value;

        fetch('/api/stats/' + ip + '?range=' + range + '&date=' + date)
            .then(r => r.json())
            .then(resp => renderDetailChart(resp.data || [], name))
            .catch(err => console.error('query detail:', err));
    }

    function renderDetailChart(data, name) {
        document.getElementById('detail-card').style.display = 'block';
        document.getElementById('detail-title').textContent = name;

        if (!detailChart) {
            detailChart = echarts.init(document.getElementById('detail-chart'));
        }

        const times = data.map(d => {
            const dt = new Date(d.timestamp * 1000);
            return dt.getHours().toString().padStart(2, '0') + ':' + dt.getMinutes().toString().padStart(2, '0');
        });
        const tx = data.map(d => d.tx_bytes);
        const rx = data.map(d => d.rx_bytes);

        detailChart.setOption({
            tooltip: {
                trigger: 'axis',
                formatter: function(params) {
                    let s = params[0].name + '<br/>';
                    params.forEach(p => {
                        s += p.marker + p.seriesName + ': ' + Utils.formatBytes(p.value) + '<br/>';
                    });
                    return s;
                }
            },
            legend: { data: ['上行', '下行'] },
            xAxis: { type: 'category', data: times },
            yAxis: {
                type: 'value',
                axisLabel: { formatter: function(v) { return Utils.formatBytes(v); } }
            },
            series: [
                { name: '上行', type: 'line', data: tx, smooth: true, itemStyle: { color: '#1a73e8' } },
                { name: '下行', type: 'line', data: rx, smooth: true, itemStyle: { color: '#34a853' } }
            ]
        });
    }

    window.History = {
        activate: function() {
            if (!chart) query();
        }
    };
})();
