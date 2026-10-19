import {contrastColor} from '../utils/color.ts';
import {createSortable} from '../modules/sortable.ts';
import {POST, DELETE, PUT} from '../modules/fetch.ts';
import {type SortableEvent} from 'sortablejs';
import {showErrorToast} from '../modules/toast.ts';

const {i18n} = window.config;

function updateIssueCount(cards: Element) {
  const parent = cards.parentElement;
  const cnt = parent.querySelectorAll('.issue-card').length;
  parent.querySelector('.project-column-issue-count').textContent = cnt.toString();
}

async function createNewColumn(url: string, columnTitle: HTMLInputElement, projectColorInput: HTMLInputElement) {
  try {
    await POST(url, {
      data: {
        title: columnTitle.value,
        color: projectColorInput.value,
      },
    });

    columnTitle.closest('form').classList.remove('dirty');
    window.location.reload();
  } catch (error) {
    console.error(error);
    showErrorToast(i18n.network_error);
  }
}

async function moveIssue({item, from, to, oldIndex}: SortableEvent) {
  const columnCards = to.querySelectorAll('.issue-card');
  updateIssueCount(from);
  updateIssueCount(to);

  const columnSorting = {
    issues: Array.from(columnCards, (card, i) => ({
      issueID: parseInt(card.getAttribute('data-issue')),
      sorting: i,
    })),
  };

  try {
    await POST(`${to.getAttribute('data-url')}/move`, {
      data: columnSorting,
    });
  } catch (error) {
    console.error(error);
    from.insertBefore(item, from.children[oldIndex]);
  }
}

async function initRepoProjectSortable() {
  const mainBoard = document.querySelector<HTMLDivElement>('#project-board > .board.sortable');
  if (!mainBoard) return;

  // the HTML layout is: #project-board > .board > .project-column .cards > .issue-card
  let boardColumns = mainBoard.querySelectorAll('.project-column');
  createSortable(mainBoard, {
    group: 'project-column',
    draggable: '.project-column',
    handle: '.project-column-header',
    delayOnTouchOnly: true,
    delay: 500,
    onSort: async () => {
      boardColumns = mainBoard.querySelectorAll('.project-column');

      const columnSorting = {
        columns: Array.from(boardColumns, (column, i) => ({
          columnID: parseInt(column.getAttribute('data-id')),
          sorting: i,
        })),
      };

      try {
        await POST(mainBoard.getAttribute('data-url'), {
          data: columnSorting,
        });
      } catch (error) {
        console.error(error);
      }
    },
  });

  for (const boardColumn of boardColumns) {
    const boardCardList = boardColumn.querySelector<HTMLDivElement>('.cards');
    createSortable(boardCardList, {
      group: 'shared',
      onAdd: moveIssue,
      onUpdate: moveIssue,
      delayOnTouchOnly: true,
      delay: 500,
    });
  }
}

export function initRepoProject() {
  if (!document.querySelector('.repository.projects')) {
    return;
  }

  initRepoProjectSortable(); // intentionally not await-ed

  for (const modal of document.querySelectorAll<HTMLDialogElement>('dialog.edit-project-column-modal')) {
    const projectHeader = modal.closest('.project-column-header');
    const projectTitleLabel = projectHeader?.querySelector('.project-column-title-label');
    const projectTitleInput = modal.querySelector<HTMLInputElement>('.project-column-title-input');
    const projectColorInput = modal.querySelector<HTMLInputElement>('#new_project_column_color');
    const boardColumn = modal.closest<HTMLDivElement>('.project-column');
    modal.querySelector<HTMLButtonElement>('.edit-project-column-button')?.addEventListener('click', async function (e) {
      e.preventDefault();
      try {
        await PUT(this.getAttribute('data-url'), {
          data: {
            title: projectTitleInput?.value,
            color: projectColorInput?.value,
          },
        });

        projectTitleLabel.textContent = projectTitleInput?.value;
        projectTitleInput.closest('form')?.classList.remove('dirty');
        const dividers = boardColumn.querySelectorAll<HTMLDivElement>(':scope > .divider');
        if (projectColorInput.value) {
          const color = contrastColor(projectColorInput.value);
          boardColumn.style.setProperty('background', projectColorInput.value, 'important');
          boardColumn.style.setProperty('color', color, 'important');
          for (const divider of dividers) {
            divider.style.setProperty('color', color);
          }
        } else {
          boardColumn.style.removeProperty('background');
          boardColumn.style.removeProperty('color');
          for (const divider of dividers) {
            divider.style.removeProperty('color');
          }
        }
        modal.close();
      } catch (error) {
        console.error(error);
        showErrorToast(i18n.network_error);
      }
    });
  }

  for (const dialog of document.querySelectorAll('.default-project-column-modal')) {
    const commitButton = dialog.querySelector<HTMLButtonElement>('button[data-confirm]');
    commitButton.addEventListener('click', confirmDefaultProjectColumn);
  }

  for (const link of document.querySelectorAll('.show-delete-project-column-modal')) {
    const deleteColumnModal = document.querySelector(link.getAttribute('data-modal'));
    const confirmDeleteButton = deleteColumnModal.querySelector<HTMLButtonElement>('button[data-confirm]');
    confirmDeleteButton.addEventListener('click', confirmDeleteProjectColumn);
  }

  document.getElementById('new_project_column_submit')?.addEventListener('click', confirmNewProjectColumn);
}

async function confirmDefaultProjectColumn(this: HTMLButtonElement, e: PointerEvent) {
  e.preventDefault();
  const postUrl = this.getAttribute('data-url');

  try {
    await POST(postUrl);
    window.location.reload(); // TODO: couldn't POST-then-reload flows like this be done in HTML?
  } catch (error) {
    console.error(error);
    showErrorToast(i18n.network_error);
  }
}

async function confirmDeleteProjectColumn(this: HTMLButtonElement, e: PointerEvent) {
  e.preventDefault();
  const deleteUrl = this.getAttribute('data-url');

  try {
    await DELETE(deleteUrl);
    window.location.reload();
  } catch (error) {
    console.error(error);
    showErrorToast(i18n.network_error);
  }
}

function confirmNewProjectColumn(this: HTMLElement, e: PointerEvent) {
  e.preventDefault();
  const columnTitle = document.querySelector<HTMLInputElement>('input#new_project_column');
  const projectColorInput = document.querySelector<HTMLInputElement>('input#new_project_column_color_picker');
  if (!columnTitle.value) {
    return;
  }
  const url = this.getAttribute('data-url');
  createNewColumn(url, columnTitle, projectColorInput);
}
