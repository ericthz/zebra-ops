/**
 * init.js - 初始化入口
 * 在所有模块加载完成后执行：注入动画样式、初始化应用、全局函数
 */

// 注入通知动画样式
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    @keyframes slideOut {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
`;
document.head.appendChild(style);

// DOM 加载完成后初始化应用
document.addEventListener('DOMContentLoaded', () => {
    new ZebraOpsApp();
    applyStaticIcons();
});

/**
 * setStatus - 全局状态设置函数
 */
function setStatus(text, state) {
    var statusText = document.getElementById('statusText');
    var dot = document.getElementById('dot');
    if (statusText) statusText.textContent = text;
    if (dot) dot.className = 'dot' + (state ? ' ' + state : '');
}
