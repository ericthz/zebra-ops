/**
 * markdown.js - Markdown 渲染模块
 * 负责 marked 初始化、安全渲染（DOMPurify）、代码高亮、HTML 转义
 */

/**
 * initMarkdown - 初始化 Markdown 渲染配置
 * 轮询等待 marked 库加载完成后配置 GFM、代码高亮
 */
ZebraOpsApp.prototype.initMarkdown = function () {
    const checkMarked = () => {
        if (typeof marked !== 'undefined') {
            try {
                marked.setOptions({
                    breaks: true,
                    gfm: true,
                    headerIds: false,
                    mangle: false
                });

                if (typeof hljs !== 'undefined') {
                    marked.setOptions({
                        highlight: function (code, lang) {
                            if (lang && hljs.getLanguage(lang)) {
                                try {
                                    return hljs.highlight(code, { language: lang }).value;
                                } catch (err) {
                                    console.error('代码高亮失败:', err);
                                }
                            }
                            return code;
                        }
                    });
                }
            } catch (e) {
                console.error('Markdown 配置失败:', e);
            }
        } else {
            setTimeout(checkMarked, 100);
        }
    };
    checkMarked();
};

/**
 * renderMarkdown - 安全渲染 Markdown 内容
 * 使用 marked 解析 + DOMPurify 清洗 XSS
 */
ZebraOpsApp.prototype.renderMarkdown = function (content) {
    if (!content) return '';

    if (typeof marked === 'undefined') {
        console.warn('marked 库未加载，使用纯文本显示');
        return this.escapeHtml(content);
    }

    try {
        const html = marked.parse(content);
        if (typeof DOMPurify !== 'undefined') {
            return DOMPurify.sanitize(html);
        }
        return html;
    } catch (e) {
        console.error('Markdown 渲染失败:', e);
        return this.escapeHtml(content);
    }
};

/**
 * highlightCodeBlocks - 对容器内代码块进行语法高亮
 */
ZebraOpsApp.prototype.highlightCodeBlocks = function (container) {
    if (typeof hljs !== 'undefined' && container) {
        try {
            container.querySelectorAll('pre code').forEach((block) => {
                if (!block.classList.contains('hljs')) {
                    hljs.highlightElement(block);
                }
            });
        } catch (e) {
            console.error('代码高亮失败:', e);
        }
    }
};

/**
 * escapeHtml - HTML 转义防 XSS
 */
ZebraOpsApp.prototype.escapeHtml = function (text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
};
