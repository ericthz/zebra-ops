/**
 * history.js - 历史记录管理模块
 * 负责对话的保存、加载、删除和 localStorage 持久化
 */

/**
 * newChat - 新建对话
 * 保存当前对话 → 清空界面 → 生成新会话ID → 重置模式
 */
ZebraOpsApp.prototype.newChat = function () {
    if (this.isStreaming) {
        this.showNotification('请等待当前对话完成后再新建对话', 'warning');
        return;
    }

    if (this.currentChatHistory.length > 0) {
        if (this.isCurrentChatFromHistory) {
            this.updateCurrentChatHistory();
        } else {
            this.saveCurrentChat();
        }
    }

    this.isStreaming = false;

    if (this.messageInput) {
        this.messageInput.value = '';
    }

    this.currentChatHistory = [];
    this.isCurrentChatFromHistory = false;

    if (this.chatMessages) {
        this.chatMessages.innerHTML = '';
    }

    this.sessionId = this.generateSessionId();
    this.currentMode = 'quick';
    this.updateUI();
    this.checkAndSetCentered();

    if (this.chatContainer) {
        this.chatContainer.style.transition = 'all 0.5s ease';
    }

    this.renderChatHistory();
};

/**
 * saveCurrentChat - 保存当前对话到历史记录
 * 自动生成标题（首条用户消息前30字符），最多保存50条
 */
ZebraOpsApp.prototype.saveCurrentChat = function () {
    if (this.currentChatHistory.length === 0) return;

    const existingIndex = this.chatHistories.findIndex(h => h.id === this.sessionId);
    if (existingIndex !== -1) {
        this.updateCurrentChatHistory();
        return;
    }

    const firstUserMessage = this.currentChatHistory.find(msg => msg.type === 'user');
    const title = firstUserMessage
        ? (firstUserMessage.content.substring(0, 30) + (firstUserMessage.content.length > 30 ? '...' : ''))
        : '新对话';

    this.chatHistories.unshift({
        id: this.sessionId,
        title: title,
        messages: [...this.currentChatHistory],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString()
    });

    if (this.chatHistories.length > 50) {
        this.chatHistories = this.chatHistories.slice(0, 50);
    }

    this.saveChatHistories();
};

/**
 * updateCurrentChatHistory - 更新当前对话的历史记录
 */
ZebraOpsApp.prototype.updateCurrentChatHistory = function () {
    if (this.currentChatHistory.length === 0) return;

    const existingIndex = this.chatHistories.findIndex(h => h.id === this.sessionId);
    if (existingIndex === -1) {
        this.saveCurrentChat();
        return;
    }

    const history = this.chatHistories[existingIndex];
    history.messages = [...this.currentChatHistory];
    history.updatedAt = new Date().toISOString();

    const firstUserMessage = this.currentChatHistory.find(msg => msg.type === 'user');
    if (firstUserMessage) {
        const newTitle = firstUserMessage.content.substring(0, 30) + (firstUserMessage.content.length > 30 ? '...' : '');
        if (history.title !== newTitle) {
            history.title = newTitle;
        }
    }

    this.saveChatHistories();
};

/**
 * loadChatHistories - 从 localStorage 加载历史对话
 */
ZebraOpsApp.prototype.loadChatHistories = function () {
    try {
        const stored = localStorage.getItem('chatHistories');
        return stored ? JSON.parse(stored) : [];
    } catch (e) {
        console.error('加载历史对话失败:', e);
        return [];
    }
};

/**
 * saveChatHistories - 将历史对话保存到 localStorage
 */
ZebraOpsApp.prototype.saveChatHistories = function () {
    try {
        localStorage.setItem('chatHistories', JSON.stringify(this.chatHistories));
    } catch (e) {
        console.error('保存历史对话失败:', e);
    }
};

/**
 * renderChatHistory - 渲染历史对话列表
 */
ZebraOpsApp.prototype.renderChatHistory = function () {
    if (!this.chatHistoryList) return;

    this.chatHistoryList.innerHTML = '';

    if (this.chatHistories.length === 0) return;

    this.chatHistories.forEach((history) => {
        const isActive = history.id === this.sessionId;
        const historyItem = document.createElement('div');
        historyItem.className = 'history-item' + (isActive ? ' active' : '');
        historyItem.dataset.historyId = history.id;

        const count = history.messages ? history.messages.length : 0;
        const updatedAt = history.updatedAt
            ? new Date(history.updatedAt).toLocaleString('zh-CN', { hour12: false, month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
            : '';

        historyItem.innerHTML = `
            <div class="history-item-content">
                <span class="history-item-title">${this.escapeHtml(history.title)}</span>
                <span class="history-item-sub">${count} 条 · ${updatedAt}</span>
            </div>
            <button class="history-item-delete" data-history-id="${history.id}" title="删除">${ic('trash-2', 13)}</button>
        `;

        historyItem.addEventListener('click', (e) => {
            if (!e.target.closest('.history-item-delete')) {
                this.loadChatHistory(history.id);
            }
        });

        const deleteBtn = historyItem.querySelector('.history-item-delete');
        deleteBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            this.deleteChatHistory(history.id);
        });

        this.chatHistoryList.appendChild(historyItem);
    });
};

/**
 * loadChatHistory - 加载指定历史对话
 */
ZebraOpsApp.prototype.loadChatHistory = function (historyId) {
    const history = this.chatHistories.find(h => h.id === historyId);
    if (!history) return;

    if (this.currentChatHistory.length > 0 && this.sessionId !== historyId) {
        if (this.isCurrentChatFromHistory) {
            this.updateCurrentChatHistory();
        } else {
            this.saveCurrentChat();
        }
    }

    this.sessionId = history.id;
    this.currentChatHistory = [...history.messages];
    this.isCurrentChatFromHistory = true;

    if (this.chatMessages) {
        this.chatMessages.innerHTML = '';
        let lastUserQuestion = '';
        history.messages.forEach(msg => {
            if (msg.type === 'user' && msg.content) lastUserQuestion = msg.content;
            this.addMessage(msg.type, msg.content, false, false, msg.type === 'assistant' ? lastUserQuestion : '', !!msg.isPlainText);
        });
    }

    this.checkAndSetCentered();
    this.renderChatHistory();
};

/**
 * deleteChatHistory - 删除指定历史对话
 */
ZebraOpsApp.prototype.deleteChatHistory = function (historyId) {
    this.chatHistories = this.chatHistories.filter(h => h.id !== historyId);
    this.saveChatHistories();
    this.renderChatHistory();

    if (this.sessionId === historyId) {
        this.currentChatHistory = [];
        if (this.chatMessages) {
            this.chatMessages.innerHTML = '';
        }
        this.sessionId = this.generateSessionId();
        this.checkAndSetCentered();
    }
};
