# 🛡️ Pre-commit Hooks Guide

Welcome to the guide for setting up and using pre-commit hooks in the Kepler Operator project! Pre-commit hooks help ensure code quality, consistency, and best practices before you push your changes. This guide will walk you through the setup and highlight all the hooks configured for this repository.

---

## ✨ Overview

Pre-commit hooks automatically check your code for common issues every time you make a commit. They help prevent mistakes, enforce standards, and save time during code reviews.

---

## ⚡ Installation & Setup

1. **Install pre-commit:**

   ```sh
   pip install pre-commit
   ```

2. **Install hooks in your local git repo:**

   ```sh
   pre-commit install
   ```

   This command sets up the hooks to run automatically on `git commit`.

3. **Run hooks on all files (optional, recommended for first-time setup):**

   ```sh
   pre-commit run --all-files
   ```

---

## 🔍 Configured Hooks

The following hooks are configured in `.pre-commit-config.yaml`:

### 1. [pre-commit-hooks](https://github.com/pre-commit/pre-commit-hooks)

- `trailing-whitespace`: Removes trailing whitespace from lines.
- `end-of-file-fixer`: Ensures files end with a newline. (Excludes `docs/api.md`)
- `check-added-large-files`: Prevents committing large files.
- `check-merge-conflict`: Detects merge conflict markers.

### 2. [yamllint](https://github.com/adrienverge/yamllint.git)

- `yamllint`: Lints YAML files (excludes `bundle`, `config`, `hack/crd`).

### 3. [markdownlint](https://github.com/igorshubovych/markdownlint-cli)

- `markdownlint`: Lints Markdown files (excludes `docs/api.md`).

### 4. [codespell](https://github.com/codespell-project/codespell)

- `codespell`: Checks for spelling mistakes. Ignores words in `.codespellignore` and Go module files.

### 5. [shellcheck](https://github.com/koalaman/shellcheck-precommit)

- `shellcheck`: Lints shell scripts with extended checks (`-x`).

### 6. [golangci-lint](https://github.com/golangci/golangci-lint)

- `golangci-lint`: Runs Go linters

### 7. [commitlint](https://github.com/alessandrojcm/commitlint-pre-commit-hook)

- `commitlint`: Ensures commit messages follow the conventional format. Runs on commit messages only.

### 8. [reuse-lint-file](https://github.com/fsfe/reuse-tool)

- `reuse-lint-file`: Checks Go files for REUSE compliance.

---

## 📝 Tips & Best Practices

- If a hook fails, follow the error message to fix the issue, then recommit.
- You can temporarily skip hooks with `git commit --no-verify` (not recommended).
- To update hooks to their latest versions, run:

  ```sh
  pre-commit autoupdate
  ```

- For more details, see the [pre-commit documentation](https://pre-commit.com/).

---

Happy committing! 🚀
