(function() {
    const container = document.getElementById('devices');
    container.innerHTML = `
        <div class="card">
            <h3>已命名设备</h3>
            <table class="data-table">
                <thead>
                    <tr>
                        <th>IP 地址</th>
                        <th>设备名称</th>
                        <th>备注</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody id="devices-body"></tbody>
            </table>
        </div>
        <div class="card">
            <h3>已发现的设备</h3>
            <p style="color:#888;font-size:13px;margin-bottom:8px;">点击 IP 快速命名，绿色表示已命名</p>
            <div class="discovered-list" id="discovered-list"></div>
        </div>
        <div class="card">
            <h3>添加/编辑设备</h3>
            <div class="form-row">
                <label>IP</label>
                <input type="text" id="dev-ip" placeholder="192.168.1.10">
            </div>
            <div class="form-row">
                <label>名称</label>
                <input type="text" id="dev-name" placeholder="张三-办公机">
            </div>
            <div class="form-row">
                <label>备注</label>
                <input type="text" id="dev-note" placeholder="301 实验室">
            </div>
            <div class="form-row">
                <label></label>
                <button class="btn" id="dev-save">保存</button>
            </div>
        </div>
    `;

    document.getElementById('dev-save').addEventListener('click', saveDevice);

    let deviceMap = {};

    function loadDevices() {
        fetch('/api/devices')
            .then(r => r.json())
            .then(resp => {
                const devices = resp.data || [];
                deviceMap = {};
                devices.forEach(d => { deviceMap[d.ip] = d; });
                renderDevices(devices);
                loadDiscovered();
            })
            .catch(err => console.error('load devices:', err));
    }

    function loadDiscovered() {
        fetch('/api/realtime')
            .then(r => r.json())
            .then(resp => renderDiscovered(resp.data || []))
            .catch(err => console.error('load discovered:', err));
    }

    function renderDevices(devices) {
        const tbody = document.getElementById('devices-body');
        if (devices.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;color:#999;padding:20px;">暂无已命名设备，点击下方发现的 IP 进行命名</td></tr>';
            return;
        }
        tbody.innerHTML = devices.map(d => `
            <tr>
                <td>${d.ip}</td>
                <td>${d.name}</td>
                <td>${d.note || '-'}</td>
                <td><button class="btn btn-sm btn-outline" onclick="Devices.edit('${d.ip}','${d.name}','${d.note || ''}')">编辑</button></td>
            </tr>
        `).join('');
    }

    function renderDiscovered(data) {
        const list = document.getElementById('discovered-list');
        const ips = data.map(d => d.ip).sort((a, b) => {
            const pa = a.split('.').map(Number);
            const pb = b.split('.').map(Number);
            for (let i = 0; i < 4; i++) {
                if (pa[i] !== pb[i]) return pa[i] - pb[i];
            }
            return 0;
        });

        list.innerHTML = ips.map(ip => {
            const named = deviceMap[ip];
            const label = named ? `${ip} (${named.name})` : ip;
            const cls = named ? 'discovered-item named' : 'discovered-item';
            return `<span class="${cls}" onclick="Devices.edit('${ip}','${named ? named.name : ''}','${named ? (named.note || '') : ''}')">${label}</span>`;
        }).join('');
    }

    function saveDevice() {
        const ip = document.getElementById('dev-ip').value.trim();
        const name = document.getElementById('dev-name').value.trim();
        const note = document.getElementById('dev-note').value.trim();

        if (!ip || !name) {
            alert('请填写 IP 和名称');
            return;
        }

        fetch('/api/devices/' + ip, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name, note: note })
        })
        .then(r => r.json())
        .then(() => {
            document.getElementById('dev-ip').value = '';
            document.getElementById('dev-name').value = '';
            document.getElementById('dev-note').value = '';
            loadDevices();
        })
        .catch(err => console.error('save device:', err));
    }

    window.Devices = {
        activate: function() { loadDevices(); },
        edit: function(ip, name, note) {
            document.getElementById('dev-ip').value = ip;
            document.getElementById('dev-name').value = name;
            document.getElementById('dev-note').value = note;
            document.getElementById('dev-name').focus();
        }
    };

    loadDevices();
})();
