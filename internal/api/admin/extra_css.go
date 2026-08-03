package admin

// extraCSS 业务级样式定制，通过 amis.WithExtraCSS 注入。
//
// 覆盖品牌色、左侧导航、暗黑模式；模板保持通用，业务方可自由调整。
const extraCSS = `
.cxd-Page-aside { border-right: 1px solid var(--borderColor, #e8e8e8) !important; }

html.dark {
    --amis-bg: #0f0f1a;
    --Page-aside-bg: #12122a;
    --Page-aside-width: 220px;
}
html.dark .cxd-Page { --Page-aside-bg: #12122a; }
html.dark .cxd-Page-aside {
    border-right-color: rgba(255,255,255,0.06) !important;
}

.cxd-Nav.cxd-Nav--stacked { padding: 8px 6px; }
.cxd-Nav.cxd-Nav--stacked .cxd-Nav-item {
    margin: 2px 6px;
    padding: 10px 14px;
    border-radius: 8px;
    transition: all 0.18s ease;
    font-size: 14px;
    color: var(--textColor, #334155);
    cursor: pointer;
}
.cxd-Nav.cxd-Nav--stacked .cxd-Nav-item.is-active {
    background: linear-gradient(135deg, rgba(99,102,241,0.12) 0%, rgba(139,92,246,0.10) 100%);
    color: #6366f1;
    font-weight: 600;
}
.cxd-Nav.cxd-Nav--stacked .cxd-Nav-item:hover:not(.is-active) {
    background: rgba(99,102,241,0.05);
    color: var(--textColor, #334155);
}
.cxd-Nav.cxd-Nav--stacked .cxd-Nav-itemIcon {
    margin-right: 10px;
    width: 20px;
    text-align: center;
    font-size: 15px;
    opacity: 0.7;
}
.cxd-Nav.cxd-Nav--stacked .cxd-Nav-item.is-active .cxd-Nav-itemIcon {
    opacity: 1;
    color: #6366f1;
}

html.dark .cxd-Nav.cxd-Nav--stacked .cxd-Nav-item { color: rgba(255,255,255,0.65); }
html.dark .cxd-Nav.cxd-Nav--stacked .cxd-Nav-item.is-active {
    background: rgba(99,102,241,0.18);
    color: #a5b4fc;
}
html.dark .cxd-Nav.cxd-Nav--stacked .cxd-Nav-item:hover:not(.is-active) {
    background: rgba(255,255,255,0.05);
    color: rgba(255,255,255,0.85);
}

.shadow-sm { transition: box-shadow 0.25s ease, transform 0.25s ease; }
.shadow-sm:hover { box-shadow: 0 6px 24px rgba(0,0,0,0.09) !important; transform: translateY(-1px); }

.rbac-welcome { margin-bottom: 20px; }
.rbac-welcome-title {
    display: block; font-size: 20px; font-weight: 700;
    color: var(--textColor, #1a1a2e); margin-bottom: 4px;
}
.rbac-welcome-desc {
    display: block; font-size: 13px; color: #8899aa;
}

.cxd-Page-toolbar {
    min-height: 52px;
    padding: 0 20px !important;
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 4px;
}
.cxd-Page-content { padding: 24px !important; }
.cxd-Page-body { padding: 0 !important; }

html.dark .cxd-Card { background: rgba(255,255,255,0.03); }
html.dark .cxd-Panel { background: rgba(255,255,255,0.02); }
html.dark .cxd-Table-table { background: transparent; }
html.dark .cxd-Table-tr:hover td { background: rgba(255,255,255,0.03); }
`
