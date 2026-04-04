(function() {
    var container = document.getElementById('devices');
    var deviceMap = {};

    container.innerHTML =
        '<div class="card">' +
            '<div class="card-header">' +
                '<h3>' +
                    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px;"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>' +
                    '已发现的设备' +
                '</h3>' +
            '</div>' +
            '<p style="color:#9ca3af;font-size:13px;margin-bottom:4px;">点击设备快速编辑，<span style="color:#3b82f6;">蓝色</span> 为未命名，<span style="color:#22c55e;">绿色</span> 为已命名</p>' +
            '<div class="discovered-list" id="discovered-list"></div>' +
        '</div>' +
        '<div class="card">' +
            '<div class="card-header">' +
                '<h3>' +
                    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px;"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>' +
                    '已命名设备' +
                '</h3>' +
            '</div>' +
            '<div class="table-wrapper">' +
                '<table class="data-table">' +
                    '<thead><tr>' +
                        '<th>IP 地址</th>' +
                        '<th>设备名称</th>' +
                        '<th>备注</th>' +
                        '<th>操作</th>' +
                    '</tr></thead>' +
                    '<tbody id="devices-body"></tbody>' +
                '</table>' +
            '</div>' +
        '</div>' +
        '<div class="card">' +
            '<div class="card-header">' +
                '<h3>' +
                    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:18px;height:18px;"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>' +
                    '添加 / 编辑设备' +
                '</h3>' +
            '</div>' +
            '<div class="form-inline">' +
                '<div class="form-group">' +
                    '<label>IP 地址</label>' +
                    '<input type="text" id="dev-ip" placeholder="192.168.1.10">' +
                '</div>' +
                '<div class="form-group">' +
                    '<label>设备名称</label>' +
                    '<input type="text" id="dev-name" placeholder="张三的电脑">' +
                '</div>' +
                '<div class="form-group">' +
                    '<label>备注</label>' +
                    '<input type="text" id="dev-note" placeholder="301 实验室">' +
                '</div>' +
            '</div>' +
            '<div style="margin-top:8px;">' +
                '<button class="btn" id="dev-save">' +
                    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:16px;height:16px;"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>' +
                    '保存' +
                '</button>' +
            '</div>' +
        '</div>';

    document.getElementById('dev-save').addEventListener('click', saveDevice);

    // Allow Enter key to submit
    ['dev-ip', 'dev-name', 'dev-note'].forEach(function(id) {
        document.getElementById(id).addEventListener('keydown', function(e) {
            if (e.key === 'Enter') saveDevice();
        });
    });

    function loadDevices() {
        fetch('/api/devices')
            .then(function(r) { return r.json(); })
            .then(function(resp) {
                var devices = resp.data || [];
                deviceMap = {};
                devices.forEach(function(d) { deviceMap[d.ip] = d; });
                renderDevices(devices);
                loadDiscovered();
            })
            .catch(function(err) { console.error('load devices:', err); });
    }

    function loadDiscovered() {
        fetch('/api/realtime')
            .then(function(r) { return r.json(); })
            .then(function(resp) { renderDiscovered(resp.data || []); })
            .catch(function(err) { console.error('load discovered:', err); });
    }

    function renderDevices(devices) {
        var tbody = document.getElementById('devices-body');
        if (devices.length === 0) {
            tbody.innerHTML =
                '<tr><td colspan="4">' +
                    '<div class="empty-state">' +
                        '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>' +
                        '<p>暂无已命名设备</p>' +
                        '<p style="font-size:13px;">点击上方已发现的设备进行命名</p>' +
                    '</div>' +
                '</td></tr>';
            return;
        }
        tbody.innerHTML = devices.map(function(d) {
            return '<tr>' +
                '<td class="mono">' + Utils.escapeHtml(d.ip) + '</td>' +
                '<td><strong>' + Utils.escapeHtml(d.name) + '</strong></td>' +
                '<td>' + Utils.escapeHtml(d.note || '-') + '</td>' +
                '<td><button class="btn btn-sm btn-outline" onclick="Devices.edit(\'' + Utils.escapeHtml(d.ip) + '\',\'' + Utils.escapeHtml(d.name) + '\',\'' + Utils.escapeHtml(d.note || '') + '\')">编辑</button></td>' +
            '</tr>';
        }).join('');
    }

    function renderDiscovered(data) {
        var list = document.getElementById('discovered-list');
        var ips = data.map(function(d) { return d.ip; }).sort(function(a, b) {
            var pa = a.split('.').map(Number);
            var pb = b.split('.').map(Number);
            for (var i = 0; i < 4; i++) {
                if (pa[i] !== pb[i]) return pa[i] - pb[i];
            }
            return 0;
        });

        if (ips.length === 0) {
            list.innerHTML = '<span style="color:#9ca3af;font-size:13px;">暂未发现任何设备</span>';
            return;
        }

        list.innerHTML = ips.map(function(ip) {
            var named = deviceMap[ip];
            var label = named ? Utils.escapeHtml(ip) + ' (' + Utils.escapeHtml(named.name) + ')' : Utils.escapeHtml(ip);
            var cls = named ? 'discovered-item named' : 'discovered-item';
            return '<span class="' + cls + '" onclick="Devices.edit(\'' + Utils.escapeHtml(ip) + '\',\'' + (named ? Utils.escapeHtml(named.name) : '') + '\',\'' + (named ? Utils.escapeHtml(named.note || '') : '') + '\')">' + label + '</span>';
        }).join('');
    }

    function saveDevice() {
        var ip = document.getElementById('dev-ip').value.trim();
        var name = document.getElementById('dev-name').value.trim();
        var note = document.getElementById('dev-note').value.trim();

        if (!ip || !name) {
            alert('请填写 IP 地址和设备名称');
            return;
        }

        var btn = document.getElementById('dev-save');
        btn.disabled = true;
        btn.textContent = '保存中...';

        fetch('/api/devices/' + encodeURIComponent(ip), {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name, note: note })
        })
        .then(function(r) { return r.json(); })
        .then(function() {
            document.getElementById('dev-ip').value = '';
            document.getElementById('dev-name').value = '';
            document.getElementById('dev-note').value = '';
            loadDevices();
        })
        .catch(function(err) {
            console.error('save device:', err);
            alert('保存失败，请重试');
        })
        .finally(function() {
            btn.disabled = false;
            btn.innerHTML =
                '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:16px;height:16px;"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>' +
                '保存';
        });
    }

    window.Devices = {
        activate: function() { loadDevices(); },
        edit: function(ip, name, note) {
            document.getElementById('dev-ip').value = ip;
            document.getElementById('dev-name').value = name;
            document.getElementById('dev-note').value = note;
            document.getElementById('dev-name').focus();
            // Scroll form into view
            document.getElementById('dev-name').scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    };

    loadDevices();
})();
