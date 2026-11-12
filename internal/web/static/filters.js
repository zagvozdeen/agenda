import { closeSelects } from './select.js';

const widthKey = 'agenda.filters.width';
const collapsedKey = 'agenda.filters.collapsed';
let width = 226;
let collapsed = false;

function readPreferences() {
  try {
    const savedWidth = localStorage.getItem(widthKey);
    if (savedWidth !== null && Number.isFinite(Number(savedWidth))) width = Math.max(200, Math.min(420, Number(savedWidth)));
    collapsed = localStorage.getItem(collapsedKey) === 'true';
  } catch {}
}

function savePreferences() {
  try {
    localStorage.setItem(widthKey, String(width));
    localStorage.setItem(collapsedKey, String(collapsed));
  } catch {}
}

function maximumWidth() {
  return Math.min(420, Math.max(200, window.innerWidth - 360));
}

export function initializeFilters(root = document) {
  const board = root.querySelector('.board');
  if (!board) return;
  board.style.setProperty('--filters-width', `${Math.min(width, maximumWidth())}px`);
  board.classList.toggle('board--filters-collapsed', collapsed);
  board.querySelector('.filters').hidden = collapsed;
  const open = root.querySelector('[data-open-filters]');
  open.hidden = !collapsed;
  const close = board.querySelector('[data-close-filters]');
  close.hidden = false;
  const separator = board.querySelector('[data-resize-filters]');
  separator.hidden = false;
  separator.setAttribute('aria-valuenow', String(Math.min(width, maximumWidth())));
  separator.setAttribute('aria-valuemax', String(maximumWidth()));
}

export function setupFilters() {
  readPreferences();
  initializeFilters();

  document.addEventListener('click', event => {
    const open = event.target.closest('[data-open-filters]');
    const close = event.target.closest('[data-close-filters]');
    if (!open && !close) return;
    collapsed = Boolean(close);
    closeSelects();
    initializeFilters();
    savePreferences();
    document.querySelector(collapsed ? '[data-open-filters]' : '[data-close-filters]')?.focus({ preventScroll: true });
  });

  document.addEventListener('pointerdown', event => {
    const separator = event.target.closest('[data-resize-filters]');
    if (!separator || event.button !== 0) return;
    event.preventDefault();
    closeSelects();
    const start = event.clientX;
    const initialWidth = separator.closest('.filters').getBoundingClientRect().width;
    separator.setPointerCapture(event.pointerId);
    separator.focus({ preventScroll: true });
    document.documentElement.classList.add('page--resizing-filters');
    const move = update => {
      width = Math.round(Math.max(200, Math.min(maximumWidth(), initialWidth + update.clientX - start)));
      initializeFilters();
    };
    const finish = () => {
      separator.removeEventListener('pointermove', move);
      separator.removeEventListener('pointerup', finish);
      separator.removeEventListener('pointercancel', finish);
      separator.removeEventListener('lostpointercapture', finish);
      document.documentElement.classList.remove('page--resizing-filters');
      savePreferences();
    };
    separator.addEventListener('pointermove', move);
    separator.addEventListener('pointerup', finish);
    separator.addEventListener('pointercancel', finish);
    separator.addEventListener('lostpointercapture', finish);
  });

  document.addEventListener('keydown', event => {
    if (!event.target.matches('[data-resize-filters]') || !['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault();
    const current = Math.min(width, maximumWidth());
    width = event.key === 'Home' ? 200 : event.key === 'End' ? maximumWidth() : Math.max(200, Math.min(maximumWidth(), current + (event.key === 'ArrowRight' ? 10 : -10)));
    closeSelects();
    initializeFilters();
    savePreferences();
  });

  window.addEventListener('resize', () => initializeFilters());
  window.addEventListener('storage', event => {
    if (event.key !== null && event.key !== widthKey && event.key !== collapsedKey) return;
    width = 226;
    readPreferences();
    closeSelects();
    initializeFilters();
  });
}
