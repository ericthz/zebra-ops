/**
 * chat.js - 消息收发模块
 * 负责快速/流式消息发送、消息 DOM 构建、流式 SSE 解析
 */

/**
 * sendMessage - 发送消息主入口
 * 根据当前模式选择快速或流式发送，发送期间锁定 UI，完成后更新历史
 */
ZebraOpsApp.prototype.sendMessage = async function () {
    let message = '';
    if (this.messageInput) {
        message = this.messageInput.value.trim();
    }

    if (!message) {
        this.showNotification('请输入消息内容', 'warning');
        return;
    }

    if (this.isStreaming) {
        this.showNotification('请等待当前对话完成', 'warning');
        return;
    }

    this.addMessage('user', message);

    if (this.messageInput) {
        this.messageInput.value = '';
    }

    // 回复气泡：未收到内容时显示「思考中...」，收到回复后替换为真实内容
    const replyEl = this.addMessage('assistant', '思考中...', true, false, message);

    this.isStreaming = true;
    this.updateUI();
    setStatus('思考中', 'busy');

    let hasError = false;
    try {
        if (this.currentMode === 'quick') {
            await this.sendQuickMessage(message, replyEl);
        } else if (this.currentMode === 'stream') {
            await this.sendStreamMessage(message, replyEl);
        }
    } catch (error) {
        console.error('发送消息失败:', error);
        // 错误消息直接用 textContent 展示，避免 renderMarkdown 误解析错误文本中的特殊字符
        if (replyEl) {
            replyEl.classList.remove('streaming');
            const messageContent = replyEl.querySelector('.message-content');
            if (messageContent) {
                messageContent.textContent = '抱歉，发送消息时出现错误：' + error.message;
            }
        }
        this.currentChatHistory.push({
            type: 'assistant',
            content: '抱歉，发送消息时出现错误：' + error.message,
            timestamp: new Date().toISOString(),
            isPlainText: true
        });
        setStatus('错误', 'error');
        hasError = true;
    } finally {
        this.isStreaming = false;
        this.updateUI();
        // 仅在未处于错误状态时复位为就绪，避免覆盖错误提示
        if (!hasError) {
            setStatus('就绪');
        }

        if (this.isCurrentChatFromHistory && this.currentChatHistory.length > 0) {
            this.updateCurrentChatHistory();
            this.renderChatHistory();
        }
    }
};

/**
 * sendQuickMessage - 快速消息（非流式），POST 请求一次性获取完整回复
 */
ZebraOpsApp.prototype.sendQuickMessage = async function (message, replyEl) {
    try {
        const response = await fetch(`${this.apiBaseUrl}/chat`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ Id: this.sessionId, Question: message })
        });

        if (!response.ok) {
            throw new Error(`HTTP错误: ${response.status}`);
        }

        const data = await response.json();

        if (data.message === 'OK' && data.data && data.data.answer) {
            this._finalizeStreamMessage(replyEl, data.data.answer);
        } else {
            throw new Error(data.message || '未知错误');
        }
    } catch (error) {
        throw error;
    }
};

/**
 * sendStreamMessage - 流式消息（SSE），实时解析事件流逐步更新内容
 */
ZebraOpsApp.prototype.sendStreamMessage = async function (message, replyEl) {
    try {
        const response = await fetch(`${this.apiBaseUrl}/chat_stream`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ Id: this.sessionId, Question: message })
        });

        if (!response.ok) {
            throw new Error(`HTTP错误: ${response.status}`);
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        let currentEvent = '';
        let fullResponse = '';
        let hasContent = false;

        try {
            while (true) {
                const { done, value } = await reader.read();

                if (done) {
                    this._finalizeStreamMessage(replyEl, fullResponse);
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
                        if (currentEvent === 'done') {
                            this._finalizeStreamMessage(replyEl, fullResponse);
                            return;
                        }
                    } else if (line.startsWith('data: ')) {
                        const data = line.substring(6);
                        if (data === '[DONE]') {
                            this._finalizeStreamMessage(replyEl, fullResponse);
                            return;
                        }

                        if (currentEvent === 'message') {
                            const chunk = data === '' ? '\n' : data;
                            if (chunk.trim() !== '') {
                                hasContent = true;
                            }
                            if (hasContent) {
                                fullResponse += chunk;
                                if (replyEl) {
                                    const messageContent = replyEl.querySelector('.message-content');
                                    messageContent.textContent = fullResponse;
                                    this.scrollToBottom();
                                }
                            }
                        }
                    }
                }
            }
        } finally {
            reader.releaseLock();
        }
    } catch (error) {
        throw error;
    }
};

/**
 * _finalizeStreamMessage - 流结束后将纯文本转为 Markdown 渲染并保存到历史
 */
ZebraOpsApp.prototype._finalizeStreamMessage = function (element, fullResponse) {
    if (element) {
        element.classList.remove('streaming');
        const messageContent = element.querySelector('.message-content');
        if (messageContent) {
            if (fullResponse && fullResponse.trim()) {
                messageContent.innerHTML = this.renderMarkdown(fullResponse);
                this.highlightCodeBlocks(messageContent);
            } else {
                messageContent.textContent = '思考中...';
            }
        }
        this.scrollToBottom();
    }
    if (fullResponse) {
        this.currentChatHistory.push({
            type: 'assistant',
            content: fullResponse,
            timestamp: new Date().toISOString()
        });
        if (this.isCurrentChatFromHistory) {
            this.updateCurrentChatHistory();
            this.renderChatHistory();
        }
    }
};

/**
 * addMessage - 添加消息到聊天界面
 * 支持用户/助手消息，自动处理 Markdown 渲染和代码高亮
 * @returns {HTMLElement} 创建的消息 DOM 元素
 */
ZebraOpsApp.prototype.addMessage = function (type, content, isStreaming = false, saveToHistory = true, q = '', isPlainText = false) {
    const isFirstMessage = this.chatMessages && this.chatMessages.querySelectorAll('.message').length === 0;

    if (!isStreaming && saveToHistory && content) {
        this.currentChatHistory.push({
            type: type,
            content: content,
            timestamp: new Date().toISOString(),
            isPlainText: isPlainText || undefined
        });
    }

    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${type}${isStreaming ? ' streaming' : ''}`;

    // 左右头像：助手（左）用 logo 图标，用户（右）用 user 图标
    const messageAvatar = document.createElement('div');
    messageAvatar.className = 'message-avatar';
    messageAvatar.innerHTML = (type === 'assistant') ? ic('logo', 24) : ic('user', 24);
    messageDiv.appendChild(messageAvatar);

    const messageContentWrapper = document.createElement('div');
    messageContentWrapper.className = 'message-content-wrapper';

    const messageContent = document.createElement('div');
    messageContent.className = 'message-content';

    if (type === 'assistant' && !isStreaming && !isPlainText) {
        messageContent.innerHTML = this.renderMarkdown(content);
        this.highlightCodeBlocks(messageContent);
    } else {
        messageContent.textContent = content;
    }

    messageContentWrapper.appendChild(messageContent);

    // 元信息行：名字 + 时间 + 操作按钮
    const metaLine = document.createElement('div');
    metaLine.className = 'meta-line';
    const nameSpan = document.createElement('span');
    nameSpan.textContent = (type === 'assistant') ? 'Zebra Ops' : '你';
    const timeSpan = document.createElement('span');
    timeSpan.textContent = this.formatTime(new Date());
    metaLine.appendChild(nameSpan);
    metaLine.appendChild(timeSpan);

    const actions = document.createElement('span');
    actions.className = 'bubble-actions';
    metaLine.appendChild(actions);
    messageContentWrapper.appendChild(metaLine);

    messageDiv.appendChild(messageContentWrapper);

    if (this.chatMessages) {
        this.chatMessages.appendChild(messageDiv);
        if (isFirstMessage && this.welcomeGreeting) {
            this.welcomeGreeting.classList.add('hidden');
        }
        this.wireActions(messageDiv, messageContent, type, q);
        this.scrollToBottom();
    }

    return messageDiv;
};

/**
 * formatTime - 格式化时间为 HH:MM:SS
 */
ZebraOpsApp.prototype.formatTime = function (date) {
    if (!date) date = new Date();
    return date.toLocaleTimeString('zh-CN', { hour12: false });
};

/**
 * wireActions - 消息操作：复制 / 赞 / 踩 / 重试（助手消息才有点赞与重试）
 */
ZebraOpsApp.prototype.wireActions = function (messageDiv, messageContent, type, q) {
    const actions = messageDiv.querySelector('.bubble-actions');
    if (!actions) return;
    if (q) messageDiv.dataset.q = q;

    const self = this;

    const copy = document.createElement('button');
    copy.className = 'icon-btn';
    copy.innerHTML = ic('copy', 12) + ' 复制';
    copy.onclick = function () {
        navigator.clipboard.writeText(messageContent.innerText).then(function () {
            self.showNotification('已复制', 'success');
        }).catch(function () {
            self.showNotification('复制失败', 'error');
        });
    };
    actions.appendChild(copy);

    if (type === 'assistant') {
        const up = document.createElement('button');
        up.className = 'icon-btn';
        up.innerHTML = ic('thumbs-up', 12) + ' 赞';
        const down = document.createElement('button');
        down.className = 'icon-btn';
        down.innerHTML = ic('thumbs-down', 12) + ' 踩';
        up.onclick = function () { self.feedback(1, up, down); };
        down.onclick = function () { self.feedback(-1, up, down); };
        actions.appendChild(up);
        actions.appendChild(down);

        const retry = document.createElement('button');
        retry.className = 'icon-btn';
        retry.innerHTML = ic('refresh-cw', 12) + ' 重试';
        retry.onclick = function () {
            if (messageDiv.dataset.q) {
                self.retryMessage(messageDiv.dataset.q);
            } else {
                self.showNotification('无可重试的问题', 'warning');
            }
        };
        actions.appendChild(retry);
    }
};

/**
 * feedback - 赞/踩反馈（本地视觉标记）
 */
ZebraOpsApp.prototype.feedback = function (rating, upBtn, downBtn) {
    upBtn.classList.toggle('on', rating === 1);
    downBtn.classList.toggle('on', rating === -1);
    downBtn.classList.toggle('bad', rating === -1);
    this.showNotification(rating === 1 ? '已点赞' : '已反馈（将辅助优化）', 'success');
};

/**
 * retryMessage - 重试：将原问题回填输入框并重新发送
 */
ZebraOpsApp.prototype.retryMessage = function (q) {
    if (this.isStreaming) {
        this.showNotification('请等待当前对话完成', 'warning');
        return;
    }
    if (this.messageInput) {
        this.messageInput.value = q;
    }
    this.sendMessage();
};

/**
 * addLoadingMessage - 添加带加载动画的消息
 */
ZebraOpsApp.prototype.addLoadingMessage = function (content) {
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message assistant';

    const messageAvatar = document.createElement('div');
    messageAvatar.className = 'message-avatar';
    messageAvatar.innerHTML = ic('logo', 24);
    messageDiv.appendChild(messageAvatar);

    const messageContentWrapper = document.createElement('div');
    messageContentWrapper.className = 'message-content-wrapper';

    const messageContent = document.createElement('div');
    messageContent.className = 'message-content loading-message-content';

    const textSpan = document.createElement('span');
    textSpan.textContent = content;

    const loadingIcon = document.createElement('span');
    loadingIcon.className = 'loading-spinner-icon';
    loadingIcon.innerHTML = `
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z" fill="currentColor" opacity="0.2"/>
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10c1.54 0 3-.36 4.28-1l-1.5-2.6C13.64 19.62 12.84 20 12 20c-4.41 0-8-3.59-8-8s3.59-8 8-8c.84 0 1.64.38 2.18 1l1.5-2.6C13 2.36 12.54 2 12 2z" fill="currentColor"/>
        </svg>
    `;

    messageContent.appendChild(textSpan);
    messageContent.appendChild(loadingIcon);
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
        this.checkAndSetCentered();
        this.scrollToBottom();
    }

    return messageDiv;
};
