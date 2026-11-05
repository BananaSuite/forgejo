import {clippie} from 'clippie';
import {throttle} from 'throttle-debounce';
import {showTemporaryTooltip} from '../modules/tippy.js';

const {copy_success, copy_error} = window.config.i18n;

function escapedGitOption(name:string, value:string) {
  if (value.includes('\n')) {
    // .toBase64 is widely available since 2025 according to MDN
    // works for emoji (unlike btoa)
    const encoded = new TextEncoder().encode(value).toBase64();
    return `${name}='{base64}${encoded}'`;
  }
  // this is meant for the shell: enclose in single quotes after escaping them
  return `${name}='${value.replaceAll("'", "'\\''")}'`;
}

export function initRepoPullAgit() {
  const formElt = document.getElementById('new-pull-agit') as HTMLFormElement;
  if (!formElt) return;

  const gitCommand = document.getElementById('git-command-line') as HTMLInputElement;

  const updateGitCommand = (target: HTMLFormElement) => {
    const data = new FormData(target);
    const getString = (name:string):string => {
      const v = data.get(name);
      if (typeof v !== 'string') {
        throw new Error(`${name} should be a string, got a ${typeof v}`);
      }
      return v;
    };
    const title = getString('title').trim();
    const description = getString('content').trim();
    const head = getString('local') || 'HEAD';
    const base = getString('base');
    const topic =
      getString('local') ||
      title
        .toLowerCase()
        .replace(/ /g, '-')
        .replace(/[^\w-]+/g, '')
        .slice(0, 32);

    const s = `git push origin ${head}:refs/for/${base}/${topic} -o ${escapedGitOption('title', title)} -o ${escapedGitOption('description', description)}`;
    gitCommand.value = s;
    return s;
  };

  const throttledUpdate = throttle(100, () => updateGitCommand(formElt));
  formElt.addEventListener('input', throttledUpdate); // for user typing
  formElt.addEventListener('change', throttledUpdate); // for the branch selector
  formElt.addEventListener('submit', async (e) => {
    e.preventDefault();
    const text = updateGitCommand(e.target as HTMLFormElement);
    const success = await clippie(text);
    showTemporaryTooltip(gitCommand, success ? copy_success : copy_error);
  });
  updateGitCommand(formElt);
}
