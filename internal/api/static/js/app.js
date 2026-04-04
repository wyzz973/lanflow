(function() {
    const tabs = document.querySelectorAll('.tab');
    const contents = document.querySelectorAll('.tab-content');

    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const target = tab.dataset.tab;

            tabs.forEach(t => t.classList.remove('active'));
            contents.forEach(c => c.classList.remove('active'));

            tab.classList.add('active');
            document.getElementById(target).classList.add('active');

            if (target === 'realtime') window.Realtime && window.Realtime.activate();
            if (target === 'history') window.History && window.History.activate();
            if (target === 'devices') window.Devices && window.Devices.activate();
        });
    });

    window.Utils = {
        formatBytes: function(bytes) {
            if (bytes === 0) return '0 B';
            const units = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(1024));
            return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + units[i];
        },
        formatRate: function(bytes) {
            return this.formatBytes(bytes) + '/s';
        }
    };
})();
