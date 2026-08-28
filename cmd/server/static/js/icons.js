/**
 * icons.js - 统一图标系统（Lucide 风格内联 SVG，ISC 协议，currentColor 随主题变色）
 *
 * 用法：
 *   - 静态节点：在 HTML 上写 data-ic="图标名" 或 data-ic-prepend="图标名"（前置），配合 data-ic-size
 *   - 动态节点：在 JS 中调用 ic('图标名', size) 生成 SVG 字符串，或 iconEl() 生成 DOM 节点
 *   - 页面初始化时调用 applyStaticIcons() 渲染所有静态图标
 */
var ICONS = {
  "plus": { v:24, b:"<path fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"1.5\" d=\"M18 12h-6m0 0H6m6 0V6m0 6v6\"/>" },
  "trash-2": { v:24, b:"<path fill=\"currentColor\" d=\"M7 21q-.825 0-1.412-.587T5 19V6H4V4h5V3h6v1h5v2h-1v13q0 .825-.587 1.413T17 21zm2-4h2V8H9zm4 0h2V8h-2z\"/>" },
  "history": { v:24, b:"<path fill=\"currentColor\" d=\"M12 21q-3.15 0-5.575-1.912T3.275 14.2q-.1-.375.15-.687t.675-.363q.4-.05.725.15t.45.6q.6 2.25 2.475 3.675T12 19q2.925 0 4.963-2.037T19 12t-2.037-4.962T12 5q-1.725 0-3.225.8T6.25 8H8q.425 0 .713.288T9 9t-.288.713T8 10H4q-.425 0-.712-.288T3 9V5q0-.425.288-.712T4 4t.713.288T5 5v1.35q1.275-1.6 3.113-2.475T12 3q1.875 0 3.513.713t2.85 1.924t1.925 2.85T21 12t-.712 3.513t-1.925 2.85t-2.85 1.925T12 21m1-9.4l2.5 2.5q.275.275.275.7t-.275.7t-.7.275t-.7-.275l-2.8-2.8q-.15-.15-.225-.337T11 11.975V8q0-.425.288-.712T12 7t.713.288T13 8z\"/>" },
  "send": { v:24, b:"<path fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"1.5\" d=\"M5.5 13L18 6m-1.75 17.5h.25a72.7 72.7 0 0 1 6.504-21.962L23.26 1L23 .74l-.538.256A72.7 72.7 0 0 1 .5 7.5v.25l5 5v7.75h.25l1.774-1.69a12 12 0 0 1 2.313-1.723z\"/>" },
  "menu": { v:24, b:"<path fill=\"currentColor\" d=\"M3 18v-2h18v2zm0-5v-2h18v2zm0-5V6h18v2z\"/>" },
  "sparkles": { v:24, b:"<path fill=\"currentColor\" d=\"m19 9l-1.25-2.75L15 5l2.75-1.25L19 1l1.25 2.75L23 5l-2.75 1.25L19 9Zm0 14l-1.25-2.75L15 19l2.75-1.25L19 15l1.25 2.75L23 19l-2.75 1.25L19 23ZM9 20l-2.5-5.5L1 12l5.5-2.5L9 4l2.5 5.5L17 12l-5.5 2.5L9 20Z\"/>" },
  "sun": { v:24, b:"<path fill=\"currentColor\" d=\"M11 5V1h2v4zm6.65 2.75l-1.375-1.375l2.8-2.875l1.4 1.425zM19 13v-2h4v2zm-8 10v-4h2v4zM6.35 7.7L3.5 4.925l1.425-1.4L7.75 6.35zm12.7 12.8l-2.775-2.875l1.35-1.35l2.85 2.75zM1 13v-2h4v2zm3.925 7.5l-1.4-1.425l2.8-2.8l.725.675l.725.7zM12 18q-2.5 0-4.25-1.75T6 12t1.75-4.25T12 6t4.25 1.75T18 12t-1.75 4.25T12 18\"/>" },
  "moon": { v:24, b:"<path fill=\"currentColor\" d=\"M14 22q-2.075 0-3.9-.788t-3.175-2.137T4.788 15.9T4 12t.788-3.9t2.137-3.175T10.1 2.788T14 2q.875 0 1.75.175t1.675.525q.3.125.45.387t.15.538q0 .225-.088.425t-.287.35q-1.75 1.375-2.7 3.375T14 12q0 2.25.925 4.25t2.7 3.35q.2.15.288.363T18 20.4q0 .275-.15.538t-.45.387q-.8.35-1.662.513T14 22\"/>" },
  "settings": { v:24, b:"<path fill=\"currentColor\" d=\"m9.25 22l-.4-3.2q-.325-.125-.612-.3t-.563-.375L4.7 19.375l-2.75-4.75l2.575-1.95Q4.5 12.5 4.5 12.338v-.675q0-.163.025-.338L1.95 9.375l2.75-4.75l2.975 1.25q.275-.2.575-.375t.6-.3l.4-3.2h5.5l.4 3.2q.325.125.613.3t.562.375l2.975-1.25l2.75 4.75l-2.575 1.95q.025.175.025.338v.674q0 .163-.05.338l2.575 1.95l-2.75 4.75l-2.95-1.25q-.275.2-.575.375t-.6.3l-.4 3.2zm2.8-6.5q1.45 0 2.475-1.025T15.55 12t-1.025-2.475T12.05 8.5q-1.475 0-2.488 1.025T8.55 12t1.013 2.475T12.05 15.5\"/>" },
  "file-text": { v:24, b:"<path fill=\"currentColor\" d=\"M8 18h8v-2H8zm0-4h8v-2H8zm-2 8q-.825 0-1.412-.587T4 20V4q0-.825.588-1.412T6 2h8l6 6v12q0 .825-.587 1.413T18 22zm7-13h5l-5-5z\"/>" },
  "braces": { v:24, b:"<path fill=\"currentColor\" d=\"M14 20v-2h3q.425 0 .713-.288T18 17v-2q0-.95.55-1.725t1.45-1.1v-.35q-.9-.325-1.45-1.1T18 9V7q0-.425-.288-.712T17 6h-3V4h3q1.25 0 2.125.875T20 7v2q0 .425.288.713T21 10h1v4h-1q-.425 0-.712.288T20 15v2q0 1.25-.875 2.125T17 20zm-7 0q-1.25 0-2.125-.875T4 17v-2q0-.425-.288-.712T3 14H2v-4h1q.425 0 .713-.288T4 9V7q0-1.25.875-2.125T7 4h3v2H7q-.425 0-.712.288T6 7v2q0 .95-.55 1.725T4 11.825v.35q.9.325 1.45 1.1T6 15v2q0 .425.288.713T7 18h3v2z\"/>" },
  "refresh-cw": { v:24, b:"<path fill=\"currentColor\" d=\"M12 20q-3.35 0-5.675-2.325T4 12t2.325-5.675T12 4q1.725 0 3.3.712T18 6.75V5q0-.425.288-.712T19 4t.713.288T20 5v5q0 .425-.288.713T19 11h-5q-.425 0-.712-.288T13 10t.288-.712T14 9h3.2q-.8-1.4-2.187-2.2T12 6Q9.5 6 7.75 7.75T6 12t1.75 4.25T12 18q1.7 0 3.113-.862t2.187-2.313q.2-.35.563-.487t.737-.013q.4.125.575.525t-.025.75q-1.025 2-2.925 3.2T12 20\"/>" },
  "copy": { v:24, b:"<path fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"1.5\" d=\"M20.829 12.861c.171-.413.171-.938.171-1.986s0-1.573-.171-1.986a2.25 2.25 0 0 0-1.218-1.218c-.413-.171-.938-.171-1.986-.171H11.1c-1.26 0-1.89 0-2.371.245a2.25 2.25 0 0 0-.984.984C7.5 9.209 7.5 9.839 7.5 11.1v6.525c0 1.048 0 1.573.171 1.986c.229.551.667.99 1.218 1.218c.413.171.938.171 1.986.171s1.573 0 1.986-.171m7.968-7.968a2.25 2.25 0 0 1-1.218 1.218c-.413.171-.938.171-1.986.171s-1.573 0-1.986-.171a2.25 2.25 0 0 0-1.218 1.218c-.171.413-.171.938-.171 1.986s0 1.573-.171 1.986a2.25 2.25 0 0 1-1.218 1.218m7.968-7.968a11.68 11.68 0 0 1-7.75 7.9l-.218.068M16.5 7.5v-.9c0-1.26 0-1.89-.245-2.371a2.25 2.25 0 0 0-.983-.984C14.79 3 14.16 3 12.9 3H6.6c-1.26 0-1.89 0-2.371.245a2.25 2.25 0 0 0-.984.984C3 4.709 3 5.339 3 6.6v6.3c0 1.26 0 1.89.245 2.371c.216.424.56.768.984.984c.48.245 1.111.245 2.372.245H7.5\"/>" },
  "square": { v:24, b:"<path fill=\"currentColor\" d=\"M6 16V8q0-.825.588-1.412T8 6h8q.825 0 1.413.588T18 8v8q0 .825-.587 1.413T16 18H8q-.825 0-1.412-.587T6 16\"/>" },
  "close": { v:24, b:"<path fill=\"currentColor\" d=\"m12 13.4l-4.9 4.9q-.275.275-.7.275t-.7-.275t-.275-.7t.275-.7l4.9-4.9l-4.9-4.9q-.275-.275-.275-.7t.275-.7t.7-.275t.7.275l4.9 4.9l4.9-4.9q.275-.275.7-.275t.7.275t.275.7t-.275.7L13.4 12l4.9 4.9q.275.275.275.7t-.275.7t-.7.275t-.7-.275z\"/>" },
  "bot": { v:24, b:"<path fill=\"currentColor\" d=\"M4 15q-1.25 0-2.125-.875T1 12t.875-2.125T4 9V7q0-.825.588-1.412T6 5h3q0-1.25.875-2.125T12 2t2.125.875T15 5h3q.825 0 1.413.588T20 7v2q1.25 0 2.125.875T23 12t-.875 2.125T20 15v4q0 .825-.587 1.413T18 21H6q-.825 0-1.412-.587T4 19zm5-2q.625 0 1.063-.437T10.5 11.5t-.437-1.062T9 10t-1.062.438T7.5 11.5t.438 1.063T9 13m6 0q.625 0 1.063-.437T16.5 11.5t-.437-1.062T15 10t-1.062.438T13.5 11.5t.438 1.063T15 13m-7 4h8v-2H8z\"/>" },
  "user": { v:24, b:"<path fill=\"currentColor\" d=\"M5.85 17.1q1.275-.975 2.85-1.537T12 15t3.3.563t2.85 1.537q.875-1.025 1.363-2.325T20 12q0-3.325-2.337-5.663T12 4T6.337 6.338T4 12q0 1.475.488 2.775T5.85 17.1M12 13q-1.475 0-2.488-1.012T8.5 9.5t1.013-2.488T12 6t2.488 1.013T15.5 9.5t-1.012 2.488T12 13m0 9q-2.075 0-3.9-.788t-3.175-2.137T2.788 15.9T2 12t.788-3.9t2.137-3.175T8.1 2.788T12 2t3.9.788t3.175 2.137T21.213 8.1T22 12t-.788 3.9t-2.137 3.175t-3.175 2.138T12 22\"/>" },
  "loading": { v:24, b:"<path fill=\"currentColor\" d=\"M12 2A10 10 0 1 0 22 12A10 10 0 0 0 12 2Zm0 18a8 8 0 1 1 8-8A8 8 0 0 1 12 20Z\" opacity=\".5\"/><path fill=\"currentColor\" d=\"M20 12h2A10 10 0 0 0 12 2V4A8 8 0 0 1 20 12Z\"><animateTransform attributeName=\"transform\" dur=\"1s\" from=\"0 12 12\" repeatCount=\"indefinite\" to=\"360 12 12\" type=\"rotate\"/></path>" },
  "chevron-down": { v:24, b:"<path fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"2\" d=\"M6 9l6 6 6-6\"/>" },
  "upload": { v:24, b:"<path fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"2\" d=\"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M17 8l-5-5-5 5M12 3v12\"/>" },
  "logo": { v:48, b:"<rect width=\"48\" height=\"48\" rx=\"11\" fill=\"#4176e6\"/><path d=\"M14 14h20l-20 20h20\" fill=\"none\" stroke=\"#fff\" stroke-width=\"5.5\" stroke-linecap=\"round\" stroke-linejoin=\"round\"/><g fill=\"none\" stroke=\"#fff\" stroke-width=\"3\" stroke-linecap=\"round\" opacity=\"0.45\"><path d=\"M6 13 12 7\"/><path d=\"M7 18 13 12\"/><path d=\"M36 41 42 35\"/><path d=\"M33 44 39 38\"/></g>" }
};

function ic(name, size) {
  var s = size || 16;
  var it = ICONS[name] || { v: 24, b: '' };
  var vb = it.v || 24;
  return '<svg class="ic" width="' + s + '" height="' + s + '" viewBox="0 0 ' + vb + ' ' + vb + '" aria-hidden="true">' + it.b + '</svg>';
}

function iconEl(name, size) {
  var d = document.createElement('span');
  d.innerHTML = ic(name, size);
  return d.firstChild;
}

function applyStaticIcons() {
  var nodes = document.querySelectorAll('[data-ic]');
  for (var i = 0; i < nodes.length; i++) {
    nodes[i].innerHTML = ic(nodes[i].getAttribute('data-ic'), parseInt(nodes[i].getAttribute('data-ic-size') || '16', 10));
  }
  nodes = document.querySelectorAll('[data-ic-prepend]');
  for (var j = 0; j < nodes.length; j++) {
    nodes[j].insertAdjacentHTML('afterbegin', ic(nodes[j].getAttribute('data-ic-prepend'), parseInt(nodes[j].getAttribute('data-ic-size') || '16', 10)));
  }
}
