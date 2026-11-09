import { setupSelects } from './select.js';
import { setupBoard } from './board.js';

document.documentElement.classList.add('page--enhanced');
setupSelects();
setupBoard();

const dates = new Intl.DateTimeFormat('ru-RU', {
  day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit',
});

document.querySelectorAll('time[data-date]').forEach(element => {
  const date = new Date(element.dateTime);
  if (!Number.isNaN(date.getTime())) element.textContent = dates.format(date);
});
