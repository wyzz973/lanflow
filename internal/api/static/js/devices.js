(function() {
    const container = document.getElementById('devices');
    container.innerHTML = `
        <div class="card">
            <h3>设备管理</h3>
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
            <h3>添加/编辑设备</h3>
            <div class="form-row">
                <label>IP</label>
                <input type="text" id="dev-ip" placeholder="192.168.1.10">
            </div>
            <div class="form-row">
                <label>名称</label>
                <input type="text" id="dev-name" placeholder="GPU-01">
            </div>
            <div class="form-row">
                <label>备注</label>
                <input type="text" id="dev-note" placeholder="lab room 301">
            </div>
            <div class="form-row">
                <label></label>
                <button id="dev-save">保存</button>
            </div>
        </div>
    `;

    document.getElementById('dev-save').addEventListener('click', saveDevice);

    function loadDevices() {
        fetch('/api/devices')
            .then(r => r.json())
            .then(resp => renderDevices(resp.data || []))
            .catch(err => console.error('load devices:', err));
    }

    function renderDevices(devices) {
        const tbody = document.getElementById('devices-body');
        tbody.innerHTML = devices.map(d => `
            <tr>
                <td>${d.ip}</td>
                <td>${d.name}</td>
                <td>${d.note}</td>
                <td><button onclick="Devices.edit('${d.ip}','${d.name}','${d.note}')">编辑</button></td>
            </tr>
        `).join('');
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
        }
    };

    loadDevices();
})();
