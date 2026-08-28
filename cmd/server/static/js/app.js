/**
 * app.js - 主入口模块
 * ZebraOpsApp 构造函数 + 会话 ID 生成
 * 功能模块拆分到独立文件：chat.js / history.js / markdown.js / ui.js / upload.js / aiops.js
 */

/**
 * ZebraOpsApp - ZebraOps 智能运维前端应用主类
 */
class ZebraOpsApp {
    /**
     * 构造函数 - 初始化应用状态和 UI
     */
    constructor() {
        this.apiBaseUrl = '/api';
        this.currentMode = 'quick';
        this.sessionId = this.generateSessionId();
        this.isStreaming = false;
        this.currentChatHistory = [];
        this.chatHistories = this.loadChatHistories();
        this.isCurrentChatFromHistory = false;

        this.initializeElements();
        this.bindEvents();
        this.updateUI();
        this.initMarkdown();
        this.checkAndSetCentered();
        this.renderChatHistory();
    }

    /**
     * generateSessionId - 生成随机会话 ID
     */
    generateSessionId() {
        return 'session_' + Math.random().toString(36).substr(2, 9) + '_' + Date.now();
    }
}
