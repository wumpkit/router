# Contributing to Wumpkit Router

Thank you for considering contributing to Wumpkit Router, the official high-level command and interaction routing framework for Wumpkit. We value community contributions to make the routing engine faster, more flexible, and idiomatic.

## Getting Started

Before contributing, ensure you have the following prerequisites set up:
- **Go**: Install the latest stable version of Go (Go 1.23+).
- **Git**: Ensure Git is installed and configured.
- **Development Environment**: Clone the repository and run `go mod tidy` to fetch dependencies.
- **Testing**: Verify the test suite runs successfully with `go test ./...`.

## Contribution Workflow

### For Small Changes
Small changes, such as typo corrections or minor documentation updates, can be submitted directly as pull requests without prior discussion.

### For Larger Changes
For significant contributions, such as new features, refactors, or performance optimizations:
1. Check the [issue tracker](https://github.com/wumpkit-go/router/issues) and [discussions](https://github.com/wumpkit-go/router/discussions) to ensure your proposed change hasn't already been reported or discussed.
2. If no existing issue or discussion covers your proposal, create a new issue or start a discussion to outline your idea and engage with maintainers and the community.

### Branching Strategy

Wumpkit Router follows a branching model inspired by Git Flow:
- **master**: Represents the production-ready, stable version of the library.
- **develop**: Integration point for all new features, fixes, and improvements.

Branch names should follow this convention:
- `feature/`: For new features (e.g., `feature/route-grouping`).
- `fix/`: For bug fixes (e.g., `fix/cooldown-race-condition`).
- `refactor/`: For code refactoring without changing functionality.
- `perf/`: For performance optimizations (e.g., `perf/matcher-zero-alloc`).
- `docs/`: For documentation changes.
- `test/`: For adding or improving tests.

### Commit Conventions

Commits must adhere to the [Conventional Commits specification](https://www.conventionalcommits.org/en/v1.0.0/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Common commit types include:
- `feat`: Introduces a new feature.
- `fix`: Addresses a bug.
- `refactor`: Modifies code without altering functionality.
- `perf`: Improves performance.
- `docs`: Updates documentation.
- `test`: Adds or improves tests.
- `chore`: Updates build processes or auxiliary tools.

### Submitting Pull Requests

1. Fork the repository and clone it locally.
2. Create a new branch: `git checkout -b feature/your-feature-name`.
3. Implement your changes and commit them using Conventional Commits.
4. Ensure code formatting is clean: `go fmt ./...`.
5. Run all tests with the race detector: `go test -race ./...`.
6. Push your branch to your fork: `git push origin feature/your-feature-name`.
7. Open a pull request targeting `master`.

### Code Style

- Adhere to Go standards and conventions.
- Keep comments concise, lowercased, and without trailing periods.
- Avoid fluent builder patterns; prioritize Go types and functional options.
- Zero-allocation fast paths are encouraged for hot-path dispatching and pattern matching.

### Licensing

By contributing to Wumpkit Router, you agree that your contributions are licensed under the [BSD 3-Clause License](LICENSE).
