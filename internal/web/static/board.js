import { initializeSelects, restoreSelect, closeSelects } from './select.js';
import { formatDates } from './dates.js';
import { initializeFilters } from './filters.js';

let controller;
let sequence = 0;
let retryURL;

function notice(message, error = false) {
  const element = document.getElementById('request-notice');
  element.replaceChildren();
  element.hidden = !message;
  element.classList.toggle('request-notice--error', error);
  element.setAttribute('role', error ? 'alert' : 'status');
  element.append(document.createTextNode(message));
  if (error) {
    const retry = document.createElement('button');
    retry.className = 'request-notice__retry';
    retry.type = 'button';
    retry.textContent = 'Retry';
    retry.dataset.retry = '';
    element.append(retry);
  }
}

function updateMetadata(workspace) {
  document.title = workspace.dataset.title;
  document.querySelector('link[rel="canonical"]').href = workspace.dataset.canonical;
  const values = {
    description: workspace.dataset.description,
    'og:title': workspace.dataset.title,
    'og:description': workspace.dataset.description,
    'og:url': workspace.dataset.canonical,
    'og:image': workspace.dataset.image,
    'og:image:alt': workspace.dataset.imageAlt,
    'twitter:title': workspace.dataset.title,
    'twitter:description': workspace.dataset.description,
    'twitter:image': workspace.dataset.image,
    'twitter:image:alt': workspace.dataset.imageAlt,
  };
  for (const [name, value] of Object.entries(values)) {
    document.querySelector(`meta[name="${name}"], meta[property="${name}"]`)?.setAttribute('content', value);
  }
}

function formURL(form) {
  const values = new URLSearchParams();
  for (const [name, value] of new FormData(form)) {
    if (value !== '' || name === 'board') values.append(name, value);
  }
  const columns = values.getAll('columns');
  if (columns.length === 4 && ['todo', 'in_progress', 'review', 'done'].every(value => columns.includes(value))) {
    values.delete('columns');
    values.delete('columns_set');
  }
  const url = new URL(form.action, location.href);
  const board = values.get('board');
  if (/^[A-Z]{3}$/.test(board)) {
    url.pathname = `/${board}`;
    values.delete('board');
  }
  url.search = values.toString();
  return url;
}

async function loadBoard(url, push = true) {
  controller?.abort();
  controller = new AbortController();
  const request = ++sequence;
  const workspace = document.getElementById('workspace');
  workspace.setAttribute('aria-busy', 'true');
  workspace.querySelector('.kanban')?.classList.add('kanban--loading');
  retryURL = url;
  const timer = setTimeout(() => { if (request === sequence) notice('Загрузка задач…'); }, 250);
  try {
    const response = await fetch(url, {
      headers: { 'X-Agenda-Partial': 'board' },
      signal: controller.signal,
      credentials: 'same-origin',
    });
    if (!response.ok || !response.headers.get('Content-Type')?.startsWith('text/html')) {
      throw new Error('Failed to load tasks');
    }
    const html = await response.text();
    if (request !== sequence) return;
    const content = new DOMParser().parseFromString(html, 'text/html');
    const replacement = content.getElementById('workspace');
    if (!replacement || replacement.dataset.page !== 'board') throw new Error('Invalid board response');
    const focus = document.activeElement?.closest('[data-select]');
    const selection = focus ? {
      name: focus.dataset.select,
      open: focus.open,
      value: focus.querySelector('[data-active]')?.dataset.value,
    } : null;
    const hadResetFocus = document.activeElement?.matches('[data-reset-filters]');
    workspace.replaceWith(replacement);
    initializeFilters(replacement);
    initializeSelects(replacement);
    formatDates(replacement);
    updateMetadata(replacement);
    if (push && url.href !== location.href) history.pushState(null, '', url);
    if (selection) restoreSelect(selection.name, selection.open, selection.value);
    else if (hadResetFocus) replacement.querySelector('[data-reset-filters]')?.focus({ preventScroll: true });
    document.getElementById('board-announcer').textContent = replacement.querySelector('.board__total').textContent;
    notice('');
  } catch (error) {
    if (request !== sequence || error.name === 'AbortError') return;
    notice('Could not load tasks. Please try again.', true);
  } finally {
    clearTimeout(timer);
    if (request === sequence) {
      const current = document.getElementById('workspace');
      current?.removeAttribute('aria-busy');
      current?.querySelector('.kanban')?.classList.remove('kanban--loading');
    }
  }
}

export function setupBoard() {
  document.addEventListener('change', event => {
    const form = event.target.closest('#filters-form');
    if (!form) return;
    if (event.target.name === 'board') {
      form.querySelectorAll('input[name="sprint"]').forEach(input => { input.checked = input.value === ''; });
      initializeSelects(form);
    }
    loadBoard(formURL(form));
  });

  document.addEventListener('submit', event => {
    if (!event.target.matches('#filters-form')) return;
    event.preventDefault();
    closeSelects();
    loadBoard(formURL(event.target));
  });

  document.addEventListener('click', event => {
    const reset = event.target.closest('[data-reset-filters]');
    if (reset && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey && event.button === 0) {
      event.preventDefault();
      closeSelects();
      loadBoard(new URL(reset.href));
    }
    if (event.target.closest('[data-retry]') && retryURL) loadBoard(retryURL);
  });

  window.addEventListener('popstate', () => {
    if ((location.pathname === '/' || /^\/[A-Z]{3}$/.test(location.pathname)) && document.getElementById('filters-form')) {
      closeSelects();
      loadBoard(new URL(location.href), false);
    }
  });
}
