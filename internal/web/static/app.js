import { setupSelects } from './select.js';
import { setupBoard } from './board.js';
import { formatDates } from './dates.js';
import { setupFilters } from './filters.js';

document.documentElement.classList.add('page--enhanced');
setupSelects();
setupFilters();
setupBoard();

formatDates();
