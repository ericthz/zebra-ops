/**
 * upload.js - 文件上传模块
 * 负责文件选择、类型验证、上传到知识库、文件大小格式化
 */

/**
 * handleFileSelect - 处理文件选择事件
 */
ZebraOpsApp.prototype.handleFileSelect = function (event) {
    const file = event.target.files[0];
    if (file) {
        if (!this.validateFileType(file)) {
            this.showNotification('只支持上传 TXT 或 Markdown (.md) 格式的文件', 'error');
            this.fileInput.value = '';
            return;
        }
        this.uploadFile(file);
    }
};

/**
 * validateFileType - 验证文件类型是否允许
 */
ZebraOpsApp.prototype.validateFileType = function (file) {
    const fileName = file.name.toLowerCase();
    const allowedExtensions = ['.txt', '.md', '.markdown'];
    return allowedExtensions.some(ext => fileName.endsWith(ext));
};

/**
 * uploadFile - 上传文件到知识库
 * 验证文件类型和大小，发送 FormData 到后端
 */
ZebraOpsApp.prototype.uploadFile = async function (file) {
    if (!this.validateFileType(file)) {
        this.showNotification('只支持上传 TXT 或 Markdown (.md) 格式的文件', 'error');
        return;
    }

    const maxSize = 50 * 1024 * 1024;
    if (file.size > maxSize) {
        this.showNotification('文件大小不能超过50MB', 'error');
        return;
    }

    this.isStreaming = true;
    this.updateUI();
    this.showUploadOverlay(true, file.name);

    try {
        const formData = new FormData();
        formData.append('file', file);

        const response = await fetch(`${this.apiBaseUrl}/upload`, {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            throw new Error(`HTTP错误: ${response.status}`);
        }

        const data = await response.json();

        if (data.message === 'OK' && data.data) {
            const successMessage = `${file.name} 上传到知识库成功`;
            this.addMessage('assistant', successMessage, false, true);
        } else {
            throw new Error(data.message || '上传失败');
        }
    } catch (error) {
        console.error('文件上传失败:', error);
        this.showNotification('文件上传失败: ' + error.message, 'error');
    } finally {
        if (this.fileInput) this.fileInput.value = '';
        this.isStreaming = false;
        this.showUploadOverlay(false);
        this.updateUI();
    }
};

/**
 * formatFileSize - 格式化文件大小显示
 */
ZebraOpsApp.prototype.formatFileSize = function (bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
};
