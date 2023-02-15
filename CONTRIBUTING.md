# Contribute to Kepler Operator

Welcome to the Kepler Operator community and thank you for contributing to Kepler Operator!
May this guide help you with your 1st contribution.

There are multiple ways to contribute, including new feature requests and implementations, bug reports and fixes, PR reviews, doc updates, refactoring, unit and integration tests, web design, etc.

1. Before opening a new issue or PR, search for any existing issues [here](https://github.com/sustainable-computing-io/kepler-operator/issues) to avoid duplication.
2. For any code contribution, please read the documents below carefully:
   -  [License](./LICENSE)
   -  [DCO](./DCO)

If you are good with our [License](./LICENSE) and [DCO](./DCO), follow these steps to start with your 1st code contribution:
1. Fork & clone Kepler Operator
2. We use  as a test framework. Please add units tests that cover your code changes.
3. For any new feature design, or feature level changes, please create an issue 1st, then submit a PR.



Here is a checklist for when you are ready to open a Pull Request:
1. Add [unit tests](#unit-tests) that cover your changes
2. Ensure that all unit tests are successful
3. Run the [integration tests](#integration-tests) locally
4. [Sign](#signed-commits) your commits
5. [Format](#commit-messages) your commit messages

Once a PR is open, Kepler Operator [reviewers](./Contributors.md) will review the PR. Thank you for contributing to Kepler Operator!

## Local Development Environment
TO-DO


## Testing

### CI Tests
The Kepler-Operator CI tests perform the following checks:
- [integration tests](./.github/workflows/integration_test.yml)

### Unit Tests
We run Go tests based on specific build tags for different conditions.
Please don't break other build tags, otherwise CI may fail.

To run the unit tests:
```
make test
```

### Integration Tests
Integration tests should be based on the miminal scope of a unit test needed to succeeded.

The GitHub Actions workflow for integration tests and in-depth steps can be found [here](./.github/workflows/integration_test.yml).

  
## Sign Commits

Please sign and commit your changes with `git commit -s`. More information on how to sign commits can be found [here](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits).

## Commit Messages
Please refer to the Kubernetes commit message guidelines that can be found [here](https://www.kubernetes.dev/docs/guide/pull-requests/#commit-message-guidelines).

We have 3 rules as commit messages check to ensure a commit message is meaningful:
- Try to keep the subject line to 50 characters or less; do not exceed 72 characters
- Providing additional context with the following formatting: `<topic>: <something>`
- The first word in the commit message subject should be capitalized unless it starts with a lowercase symbol or other identifier

For example:
```
Doc: update Developer.md
```

