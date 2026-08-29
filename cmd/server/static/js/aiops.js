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
 * sendAIOpsRequest - 通过 SSE 流式接收 AI 运维分析进度与最终报告
 */
ZebraOpsApp.prototype.sendAIOpsRequest = async function (loadingMessageElement) {
    try {
        const response = await fetch(`${this.apiBaseUrl}/ai_ops_stream`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ Id: this.sessionId })
        });

        if (!response.ok) {
            throw new Error(`HTTP错误: ${response.status}`);
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        let currentEvent = '';
        let details = [];

        try {
            while (true) {
                const { done, value } = await reader.read();
                if (done) {
                    break;
                }

                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop() || '';

                for (const line of lines) {
                    if (line.trim() === '') continue;

                    if (line.startsWith('id: ')) {
                        continue;
                    } else if (line.startsWith('event: ')) {
                        currentEvent = line.substring(7);
                    } else if (line.startsWith('data: ')) {
                        const data = line.substring(6);
                        if (currentEvent === 'status') {
                            this._updateAIOpsProgress(loadingMessageElement, data);
                        } else if (currentEvent === 'step') {
                            details.push(this._extractText(data));
                            this._updateAIOpsProgress(loadingMessageElement, data);
                        } else if (currentEvent === 'done') {
                            const payload = JSON.parse(data);
                            const report = payload.report || '';
                            this.updateAIOpsMessage(loadingMessageElement, report, details);
                            return;
                        } else if (currentEvent === 'error') {
                            throw new Error(this._extractText(data));
                        }
                    }
                }
            }
        } finally {
            reader.releaseLock();
        }

        // 流异常结束（无 done 事件）视为失败
        throw new Error('分析流意外中断，未收到最终报告');
    } catch (error) {
        throw error;
    }
};

/**
 * _extractText - 从 SSE data（JSON 对象）中提取 text 字段，失败时返回原文
 */
ZebraOpsApp.prototype._extractText = function (data) {
    try {
        return JSON.parse(data).text || '';
    } catch (e) {
        return data;
    }
};

/**
 * _updateAIOpsProgress - 更新加载消息上的进度文案（显示当前执行步骤）
 */
ZebraOpsApp.prototype._updateAIOpsProgress = function (messageElement, data) {
    if (!messageElement) return;
    const messageContent = messageElement.querySelector('.message-content');
    if (!messageContent) return;

    let text = this._extractText(data);
    if (!text) return;

    // 步骤明细较长时仅展示首行并截断，避免气泡过长
    const firstLine = text.split('\n')[0].trim();
    const shortText = firstLine.length > 60 ? firstLine.substring(0, 60) + '…' : firstLine;

    const textSpan = messageContent.querySelector('span');
    if (textSpan) {
        textSpan.textContent = '分析中... ' + shortText;
    } else {
        messageContent.textContent = '分析中... ' + shortText;
    }
    this.scrollToBottom();
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
