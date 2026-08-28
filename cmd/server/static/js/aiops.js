/**
 * aiops.js - AI 运维分析模块
 * 负责触发分析、调用后端接口、渲染分析报告与可折叠步骤明细
 */

/**
 * triggerAIOps - 触发智能运维分析
 * 新建对话 → 显示加载动画 → 调用后端 → 更新消息
 */
ZebraOpsApp.prototype.triggerAIOps = async function () {
    if (this.isStreaming) {
        this.showNotification('请等待当前操作完成', 'warning');
        return;
    }

    this.newChat();

    const loadingMessage = this.addLoadingMessage('分析中...');
    this.currentAIOpsMessage = loadingMessage;

    this.isStreaming = true;
    this.updateUI();
    setStatus('分析中', 'busy');

    let hasError = false;
    try {
        await this.sendAIOpsRequest(loadingMessage);
    } catch (error) {
        console.error('AI 运维分析失败:', error);
        setStatus('错误', 'error');
        hasError = true;
        if (loadingMessage) {
            const messageContent = loadingMessage.querySelector('.message-content');
            if (messageContent) {
                messageContent.classList.remove('loading-message-content');
                // 错误消息直接用 textContent 展示，避免 renderMarkdown 误解析错误文本中的特殊字符
                const errText = '抱歉，AI 运维分析时出现错误：' + error.message;
                messageContent.textContent = errText;
            }
        }
    } finally {
        this.isStreaming = false;
        this.currentAIOpsMessage = null;
        this.updateUI();
        // 仅在未处于错误状态时复位为就绪，避免覆盖错误提示
        if (!hasError) {
            setStatus('就绪');
        }
    }
};

/**
 * sendAIOpsRequest - 发送 AI 运维请求到后端
 */
ZebraOpsApp.prototype.sendAIOpsRequest = async function (loadingMessageElement) {
    try {
        const response = await fetch(`${this.apiBaseUrl}/ai_ops`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ Id: this.sessionId })
        });

        if (!response.ok) {
            throw new Error(`HTTP错误: ${response.status}`);
        }

        const data = await response.json();

        if (data.message === 'OK' && data.data) {
            const responseText = data.data.result || '';
            this.updateAIOpsMessage(loadingMessageElement, responseText, data.data.detail || []);
        } else {
            throw new Error(data.message || '未知错误');
        }
    } catch (error) {
        throw error;
    }
};

/**
 * updateAIOpsMessage - 将加载动画替换为分析结果
 * 包含可折叠的详细步骤
 */
ZebraOpsApp.prototype.updateAIOpsMessage = function (messageElement, response, details) {
    if (!messageElement) {
        return this.addAIOpsMessage(response, details);
    }

    messageElement.classList.add('aiops-message');

    const messageContentWrapper = messageElement.querySelector('.message-content-wrapper');
    if (!messageContentWrapper) return;

    const messageContent = messageContentWrapper.querySelector('.message-content');
    if (!messageContent) return;

    messageContent.classList.remove('loading-message-content');
    messageContent.textContent = '';

    const loadingIcon = messageContent.querySelector('.loading-spinner-icon');
    if (loadingIcon) loadingIcon.remove();

    // 可折叠详情
    if (details && details.length > 0) {
        let detailsContainer = messageElement.querySelector('.aiops-details');
        if (!detailsContainer) {
            detailsContainer = document.createElement('div');
            detailsContainer.className = 'aiops-details';
            messageContentWrapper.insertBefore(detailsContainer, messageContent);
        } else {
            detailsContainer.innerHTML = '';
        }

        const detailsToggle = document.createElement('div');
        detailsToggle.className = 'details-toggle';
        detailsToggle.innerHTML = `
            <svg class="toggle-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M9 18L15 12L9 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            <span>查看详细步骤 (${details.length}条)</span>
        `;

        const detailsContent = document.createElement('div');
        detailsContent.className = 'details-content';

        details.forEach((detail, index) => {
            const detailItem = document.createElement('div');
            detailItem.className = 'detail-item';
            detailItem.innerHTML = `<strong>步骤 ${index + 1}:</strong> ${this.escapeHtml(detail)}`;
            detailsContent.appendChild(detailItem);
        });

        detailsToggle.addEventListener('click', () => {
            detailsContent.classList.toggle('expanded');
            detailsToggle.classList.toggle('expanded');
        });

        detailsContainer.appendChild(detailsToggle);
        detailsContainer.appendChild(detailsContent);
    }

    messageContent.innerHTML = this.renderMarkdown(response);
    this.highlightCodeBlocks(messageContent);

    this.currentChatHistory.push({
        type: 'assistant',
        content: response,
        timestamp: new Date().toISOString()
    });

    // 持久化到本地历史（localStorage），刷新后仍可在侧栏恢复该会话
    if (this.isCurrentChatFromHistory) {
        this.updateCurrentChatHistory();
    } else {
        this.saveCurrentChat();
    }
    // AI Ops 会话无用户消息，默认标题为「新对话」，改为更可读的标题
    const aiOpsIdx = this.chatHistories.findIndex(h => h.id === this.sessionId);
    if (aiOpsIdx !== -1 && this.chatHistories[aiOpsIdx].title === '新对话') {
        this.chatHistories[aiOpsIdx].title = 'AI 运维分析';
        this.saveChatHistories();
    }
    this.renderChatHistory();

    this.scrollToBottom();
    return messageElement;
};

/**
 * addAIOpsMessage - 创建带折叠详情的 AI 运维消息
 */
ZebraOpsApp.prototype.addAIOpsMessage = function (response, details) {
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message assistant aiops-message';

    const messageAvatar = document.createElement('div');
    messageAvatar.className = 'message-avatar';
    messageAvatar.innerHTML = ic('logo', 24);
    messageDiv.appendChild(messageAvatar);

    const messageContentWrapper = document.createElement('div');
    messageContentWrapper.className = 'message-content-wrapper';

    if (details && details.length > 0) {
        const detailsContainer = document.createElement('div');
        detailsContainer.className = 'aiops-details';

        const detailsToggle = document.createElement('div');
        detailsToggle.className = 'details-toggle';
        detailsToggle.innerHTML = `
            <svg class="toggle-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M9 18L15 12L9 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            <span>查看详细步骤 (${details.length}条)</span>
        `;

        const detailsContent = document.createElement('div');
        detailsContent.className = 'details-content';

        details.forEach((detail, index) => {
            const detailItem = document.createElement('div');
            detailItem.className = 'detail-item';
            detailItem.innerHTML = `<strong>步骤 ${index + 1}:</strong> ${this.escapeHtml(detail)}`;
            detailsContent.appendChild(detailItem);
        });

        detailsToggle.addEventListener('click', () => {
            detailsContent.classList.toggle('expanded');
            detailsToggle.classList.toggle('expanded');
        });

        detailsContainer.appendChild(detailsToggle);
        detailsContainer.appendChild(detailsContent);
        messageContentWrapper.appendChild(detailsContainer);
    }

    const messageContent = document.createElement('div');
    messageContent.className = 'message-content';
    messageContent.innerHTML = this.renderMarkdown(response);
    this.highlightCodeBlocks(messageContent);
    messageContentWrapper.appendChild(messageContent);

    // 元信息行：名字 + 时间
    const metaLine = document.createElement('div');
    metaLine.className = 'meta-line';
    const nameSpan = document.createElement('span');
    nameSpan.textContent = 'Zebra Ops';
    const timeSpan = document.createElement('span');
    timeSpan.textContent = this.formatTime(new Date());
    metaLine.appendChild(nameSpan);
    metaLine.appendChild(timeSpan);
    messageContentWrapper.appendChild(metaLine);

    messageDiv.appendChild(messageContentWrapper);

    if (this.chatMessages) {
        this.chatMessages.appendChild(messageDiv);
        this.scrollToBottom();
    }

    return messageDiv;
};
