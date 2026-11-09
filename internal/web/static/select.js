let search = '';
let searchTimer;
let searchSelect;

function options(select) {
  return [...select.querySelectorAll('[data-option]')];
}

function trigger(select) {
  return select.querySelector('.select__trigger');
}

function activate(select, index, scroll = true) {
  const items = options(select);
  if (!items.length) return;
  index = Math.max(0, Math.min(items.length - 1, index));
  items.forEach((option, position) => {
    option.toggleAttribute('data-active', position === index);
    option.classList.toggle('select__option--active', position === index);
  });
  trigger(select).setAttribute('aria-activedescendant', items[index].id);
  if (scroll) items[index].scrollIntoView({ block: 'nearest' });
}

function position(select) {
  const menu = select.querySelector('.select__menu');
  const control = trigger(select).getBoundingClientRect();
  const below = window.innerHeight - control.bottom - 16;
  const above = control.top - 16;
  const upward = below < Math.min(menu.scrollHeight, 300) && above > below;
  select.classList.toggle('select--upward', upward);
  menu.style.maxHeight = `${Math.max(80, Math.min(360, upward ? above : below))}px`;
  menu.style.setProperty('--select-shift', '0px');
  const rect = menu.getBoundingClientRect();
  const shift = Math.min(0, window.innerWidth - 16 - rect.right);
  menu.style.setProperty('--select-shift', `${Math.max(16 - rect.left, shift)}px`);
}

function sync(select) {
  const selected = options(select).filter(option => option.querySelector('input').checked);
  options(select).forEach(option => {
    const checked = option.querySelector('input').checked;
    option.setAttribute('aria-selected', String(checked));
    option.classList.toggle('select__option--selected', checked);
  });
  select.querySelector('.select__value').textContent = select.hasAttribute('data-multiple')
    ? `${selected.length} из ${options(select).length}`
    : selected[0]?.querySelector('.select__option-label').textContent || '—';
}

export function closeSelects(except) {
  document.querySelectorAll('[data-select][open]').forEach(select => {
    if (select !== except) select.open = false;
  });
}

export function openSelect(select, index) {
  closeSelects(select);
  select.open = true;
  const items = options(select);
  const selected = items.findIndex(option => option.querySelector('input').checked);
  activate(select, index ?? Math.max(0, selected), false);
  trigger(select).setAttribute('aria-expanded', 'true');
  position(select);
}

export function initializeSelects(root = document) {
  root.querySelectorAll('[data-select]').forEach(select => {
    sync(select);
    trigger(select).setAttribute('aria-expanded', String(select.open));
  });
}

export function restoreSelect(name, open, value) {
  const select = [...document.querySelectorAll('[data-select]')].find(item => item.dataset.select === name);
  if (!select) return;
  if (open) {
    const index = options(select).findIndex(option => option.dataset.value === value);
    openSelect(select, index < 0 ? undefined : index);
  }
  trigger(select).focus({ preventScroll: true });
}

export function setupSelects() {
  document.addEventListener('toggle', event => {
    const select = event.target;
    if (!(select instanceof HTMLDetailsElement) || !select.matches('[data-select]')) return;
    trigger(select).setAttribute('aria-expanded', String(select.open));
    if (select.open) {
      closeSelects(select);
      position(select);
      if (!select.querySelector('[data-active]')) {
        activate(select, Math.max(0, options(select).findIndex(option => option.querySelector('input').checked)), false);
      }
    } else {
      trigger(select).removeAttribute('aria-activedescendant');
      options(select).forEach(option => {
        option.removeAttribute('data-active');
        option.classList.remove('select__option--active');
      });
    }
  }, true);

  document.addEventListener('pointerdown', event => {
    const select = event.target.closest('[data-select]');
    closeSelects(select);
  });

  document.addEventListener('change', event => {
    const select = event.target.closest('[data-select]');
    if (!select) return;
    sync(select);
    if (!select.hasAttribute('data-multiple')) select.open = false;
    trigger(select).focus({ preventScroll: true });
  });

  document.addEventListener('pointermove', event => {
    const option = event.target.closest('[data-option]');
    if (!option) return;
    const select = option.closest('[data-select]');
    if (select.open) activate(select, options(select).indexOf(option), false);
  });

  document.addEventListener('keydown', event => {
    const select = event.target.closest('[data-select]');
    if (!select || !event.target.matches('.select__trigger')) return;
    const items = options(select);
    let index = items.findIndex(option => option.hasAttribute('data-active'));
    if (event.key === 'Escape') {
      if (select.open) {
        event.preventDefault();
        select.open = false;
      }
      return;
    }
    if (event.key === 'Tab') {
      select.open = false;
      return;
    }
    if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
      event.preventDefault();
      if (!select.open) {
        openSelect(select, event.key === 'Home' ? 0 : event.key === 'End' ? items.length - 1 : undefined);
      } else {
        if (event.key === 'Home') index = 0;
        else if (event.key === 'End') index = items.length - 1;
        else index += event.key === 'ArrowDown' ? 1 : -1;
        activate(select, index);
      }
      return;
    }
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      if (!select.open) openSelect(select);
      else items[Math.max(0, index)]?.querySelector('input').click();
      return;
    }
    if (event.key.length === 1 && !event.metaKey && !event.ctrlKey && !event.altKey) {
      event.preventDefault();
      clearTimeout(searchTimer);
      if (searchSelect !== select) search = '';
      searchSelect = select;
      search += event.key.toLocaleLowerCase('ru');
      searchTimer = setTimeout(() => { search = ''; }, 650);
      if (!select.open) openSelect(select);
      const found = items.findIndex(option => option.querySelector('.select__option-label').textContent.trim().toLocaleLowerCase('ru').startsWith(search));
      if (found >= 0) activate(select, found);
    }
  });

  window.addEventListener('resize', () => {
    document.querySelectorAll('[data-select][open]').forEach(position);
  });

  initializeSelects();
}
