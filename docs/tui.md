# The commit TUI

**English** · [Español](tui.es.md)

Running `co-change` with no arguments opens a terminal tool to write [Conventional Commits](https://www.conventionalcommits.org/) in one line. While you mark files, it shows which other files usually change with them. It predates the CI focus (ADR-0019): it keeps working but gets no new features.

![co-change TUI: mark a file and its related files show up](../assets/tui.gif)

## The TUI — commit in one line

| Action | Keyboard | Mouse |
|---|---|---|
| Mark / unmark a file | `space` | click the file |
| Move through the list | `j` / `k` or arrows | — |
| Pick the type (`feat`, `fix`, …) | `tab` + its initial | click `[ feat ▾ ]` |
| Type the message and commit | type + `enter` | click the message |
| Go back / quit | `esc` · `q` · `ctrl+c` | — |

Below the commit line you see the exact subject that will be created (`→ feat: … .`), trailing period included. co-change runs `git add` **only on the marked files**, never `git add .`.

The session stays open after committing: the file list and the git log reload every 2 s, so editor changes, `git add` or commits made from another terminal show up on their own. Your marks are kept.

### Related files

When you mark files, the right column shows the five that most often change with them:

- yellow `+`: **modified and not marked**. You are probably forgetting it.
- grey `·`: usually changes with the marked files, but you have not touched it.
- `↔` in the file list: the same hint, next to the file.

The percentage is the share of the marked file's commits that also touched the other one. The index is created on first use (a few seconds in the background on a large repository) and kept up to date after every commit, checkout or rebase. If anything fails, the panel collapses to one grey line and you can still commit.

Below 80 columns the right column hides, so it never gets in the way of the minimal flow.

---

## Commands inside the TUI — `/`

Type `/` on an empty message (or click `/ comandos`) to open the palette, type a few letters and press `enter`. `esc` takes you back to the commit view. You can run co-change from any subdirectory of the repository.

### Review your changes — `/changes`

![/changes: the changed files and the selected diff](../assets/changes.png)

The changed files on the left, the whole selected file on the right with additions in green and deletions in red. Built to review what an agent or a long session changed without opening an IDE.

| Action | Keyboard | Mouse |
|---|---|---|
| Filter files by path | `/` and type | click the search box |
| Change file / scroll the diff | `j` / `k` (`tab` switches pane) | click · wheel |
| Next / previous change | `n` / `p` | the colored marks on the scrollbar |
| Jump half a page | `ctrl+d` / `ctrl+u` | drag the scrollbar |
| Mark the file for the commit | `space` | — |
| Edit the file at that change | `e` | — |

`e` hands the terminal to your editor, opened at the change you are looking at, and co-change reloads the diff when you quit. The editor is `$VISUAL` or `$EDITOR` (e.g. `export EDITOR=vim` in your `.zshrc`), else `micro`, `nano` or `vi`. co-change never edits files itself.

### Your commits by type — `/stats`

![/stats: commits by type and daily activity](../assets/stats.png)

The current branch's commits by conventional type and their activity: per day up to 30 days, per week beyond. `d` or a click on the title cycles 7 · 30 · 90 · 365 days. Subjects without a `type:` prefix count as `sin tipo` — GitHub's default squash title (`Feat/audit (#13)`) is the usual culprit, so give your PRs a conventional title.

### Files that change the most — `/audit`

![/audit: files ranked by number of commits](../assets/audit.png)

Files ranked by **how many commits touched them** (Adam Tornhill's hotspots); lines changed are context only. Files over the `/settings` threshold get `refactor?`, close to it `vigilar`. `d` cycles the window. Lockfiles and build output are skipped; add your own patterns to a `.auditignore` at the root, same format as `.cochangeignore`.

### Switch things off — `/settings`

The git-log panel, the related-files panel, mouse capture (off = native terminal selection and scroll), the log scope (current branch or all), the auto-reload interval and the `/audit` defaults. Changes apply live and are saved to `~/.config/co-change/settings.toml`.
