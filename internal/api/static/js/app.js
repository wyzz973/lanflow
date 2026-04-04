(function() {
    // Tab routing
    var tabs = document.querySelectorAll('.tab');
    var contents = document.querySelectorAll('.tab-content');

    tabs.forEach(function(tab) {
        tab.addEventListener('click', function() {
            var target = tab.dataset.tab;

            tabs.forEach(function(t) { t.classList.remove('active'); });
            contents.forEach(function(c) { c.classList.remove('active'); });

            tab.classList.add('active');
            document.getElementById(target).classList.add('active');

            if (target === 'realtime' && window.Realtime) window.Realtime.activate();
            if (target === 'history' && window.History) window.History.activate();
            if (target === 'devices' && window.Devices) window.Devices.activate();
        });
    });

    // Shared utilities
    window.Utils = {
        formatBytes: function(bytes) {
            if (!bytes || bytes === 0) return '0 B';
            var units = ['B', 'KB', 'MB', 'GB', 'TB'];
            var i = Math.floor(Math.log(Math.abs(bytes)) / Math.log(1024));
            if (i >= units.length) i = units.length - 1;
            return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + units[i];
        },

        formatRate: function(bytes) {
            return this.formatBytes(bytes) + '/s';
        },

        escapeHtml: function(str) {
            if (!str) return '';
            return String(str)
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/"/g, '&quot;');
        },

        todayStr: function() {
            var d = new Date();
            var mm = String(d.getMonth() + 1).padStart(2, '0');
            var dd = String(d.getDate()).padStart(2, '0');
            return d.getFullYear() + '-' + mm + '-' + dd;
        }
    };
})();
