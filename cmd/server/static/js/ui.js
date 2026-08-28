/**
 * ui.js - UI 控件模块
 * 负责 DOM 元素引用、事件绑定、模式切换、菜单控制、通知、遮罩层
 */

/**
 * initializeElements - 初始化所有 DOM 元素引用
 */
ZebraOpsApp.prototype.initializeElements = function () {
    this.sidebar = document.querySelector('.sidebar');
    this.newChatBtn = document.getElementById('newChatBtn');
    this.aiOpsSidebarBtn = document.getElementById('aiOpsSidebarBtn');
    this.uploadTopBtn = document.getElementById('uploadTopBtn');

    this.messageInput = document.getElementById('messageInput');
    this.sendButton = document.getElementById('sendButton');
    this.modeSelectorBtn = document.getElementById('modeSelectorBtn');
    this.modeDropdown = document.getElementById('modeDropdown');
    this.currentModeText = document.getElementById('currentModeText');
    this.fileInput = document.getElementById('fileInput');

    this.chatMessages = document.getElementById('chatMessages');
    this.loadingOverlay = document.getElementById('loadingOverlay');
    this.chatContainer = document.querySelector('main');
    this.welcomeGreeting = document.getElementById('welcomeGreeting');
    this.chatHistoryList = document.getElementById('chatHistoryList');

    this.checkAndSetCentered();
};

/**
 * bindEvents - 绑定所有事件监听器
 */
ZebraOpsApp.prototype.bindEvents = function () {
    if (this.newChatBtn) {
        this.newChatBtn.addEventListener('click', () => this.newChat());
    }

    if (this.aiOpsSidebarBtn) {
        this.aiOpsSidebarBtn.addEventListener('click', () => this.triggerAIOps());
    }

    if (this.uploadTopBtn) {
        this.uploadTopBtn.addEventListener('click', () => {
            if (this.fileInput) this.fileInput.click();
        });
    }

    if (this.modeSelectorBtn) {
        this.modeSelectorBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            this.toggleModeDropdown();
        });
    }

    const dropdownItems = document.querySelectorAll('.dropdown-item');
    dropdownItems.forEach(item => {
        item.addEventListener('click', () => {
            this.selectMode(item.getAttribute('data-mode'));
            this.closeModeDropdown();
        });
    });

    document.addEventListener('click', (e) => {
        if (!this.modeSelectorBtn.contains(e.target) &&
            !this.modeDropdown.contains(e.target)) {
            this.closeModeDropdown();
        }
    });

    if (this.sendButton) {
        this.sendButton.addEventListener('click', () => this.sendMessage());
    }

    if (this.messageInput) {
        this.messageInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendMessage();
            }
        });
    }

    if (this.fileInput) {
        this.fileInput.addEventListener('change', (e) => this.handleFileSelect(e));
    }
};

/**
 * toggleModeDropdown - 切换模式下拉菜单
 */
ZebraOpsApp.prototype.toggleModeDropdown = function () {
    if (this.modeSelectorBtn && this.modeDropdown) {
        const wrapper = this.modeSelectorBtn.closest('.mode-selector-wrapper');
        if (wrapper) wrapper.classList.toggle('active');
    }
};

/**
 * closeModeDropdown - 关闭模式下拉菜单
 */
ZebraOpsApp.prototype.closeModeDropdown = function () {
    if (this.modeSelectorBtn && this.modeDropdown) {
        const wrapper = this.modeSelectorBtn.closest('.mode-selector-wrapper');
        if (wrapper) wrapper.classList.remove('active');
    }
};

/**
 * selectMode - 选择对话模式
 */
ZebraOpsApp.prototype.selectMode = function (mode) {
    if (this.isStreaming) {
        this.showNotification('请等待当前对话完成后再切换模式', 'warning');
        return;
    }

    this.currentMode = mode;
    this.updateUI();

    const modeNames = { 'quick': '快速', 'stream': '流式' };
    this.showNotification(`已切换到${modeNames[mode]}模式`, 'info');
};

/**
 * updateUI - 同步界面状态
 */
ZebraOpsApp.prototype.updateUI = function () {
    if (this.currentModeText) {
        const modeNames = { 'quick': '快速', 'stream': '流式' };
        this.currentModeText.textContent = modeNames[this.currentMode] || '快速';
    }

    const dropdownItems = document.querySelectorAll('.dropdown-item');
    dropdownItems.forEach(item => {
        const mode = item.getAttribute('data-mode');
        item.classList.toggle('active', mode === this.currentMode);
    });

    if (this.sendButton) {
        this.sendButton.disabled = this.isStreaming;
    }

    if (this.messageInput) {
        this.messageInput.disabled = this.isStreaming;
        this.messageInput.placeholder = '问问 Zebra Ops AI 助手';
    }
};

/**
 * showNotification - 显示通知提示（3秒自动消失）
 */
ZebraOpsApp.prototype.showNotification = function (message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.textContent = message;

    const colors = { info: '#1a73e8', success: '#34a853', warning: '#fbbc04', error: '#ea4335' };
    notification.style.backgroundColor = colors[type] || colors.info;

    document.body.appendChild(notification);

    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => {
            if (notification.parentNode) notification.parentNode.removeChild(notification);
        }, 300);
    }, 3000);
};

/**
 * showLoadingOverlay - 显示/隐藏 AI 运维加载遮罩
 */
ZebraOpsApp.prototype.showLoadingOverlay = function (show) {
    if (this.loadingOverlay) {
        if (show) {
            this.loadingOverlay.style.display = 'flex';
            const loadingText = this.loadingOverlay.querySelector('.loading-text');
            const loadingSubtext = this.loadingOverlay.querySelector('.loading-subtext');
            if (loadingText) loadingText.textContent = 'AI 运维分析中，请稍候...';
            if (loadingSubtext) loadingSubtext.textContent = '后端正在处理，请耐心等待';
            document.body.style.overflow = 'hidden';
        } else {
            this.loadingOverlay.style.display = 'none';
            document.body.style.overflow = '';
        }
    }
};

/**
 * showUploadOverlay - 显示/隐藏文件上传遮罩
 */
ZebraOpsApp.prototype.showUploadOverlay = function (show, fileName = '') {
    if (this.loadingOverlay) {
        if (show) {
            this.loadingOverlay.style.display = 'flex';
            const loadingText = this.loadingOverlay.querySelector('.loading-text');
            const loadingSubtext = this.loadingOverlay.querySelector('.loading-subtext');
            if (loadingText) loadingText.textContent = '正在上传文件...';
            if (loadingSubtext) loadingSubtext.textContent = fileName ? `上传: ${fileName}` : '请稍候';
            document.body.style.overflow = 'hidden';
        } else {
            this.loadingOverlay.style.display = 'none';
            document.body.style.overflow = '';
        }
    }
};

/**
 * checkAndSetCentered - 检查并设置欢迎语可见性
 */
ZebraOpsApp.prototype.checkAndSetCentered = function () {
    if (this.chatMessages && this.welcomeGreeting) {
        const hasMessages = this.chatMessages.querySelectorAll('.message').length > 0;
        this.welcomeGreeting.classList.toggle('hidden', hasMessages);
    }
};

/**
 * scrollToBottom - 滚动聊天区域到底部（滚动容器为主滚动区 main）
 */
ZebraOpsApp.prototype.scrollToBottom = function (force) {
    const scroller = this.chatContainer || this.chatMessages;
    if (!scroller) return;
    if (force === true) {
        scroller.scrollTop = scroller.scrollHeight;
        return;
    }
    const near = scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight < 120;
    if (near) scroller.scrollTop = scroller.scrollHeight;
};
