## Description

<!-- Provide a brief description of the changes in this PR -->

## Related Issue

<!-- Link to the related issue(s), e.g., Fixes #123 or Closes #456 -->

## Type of Change

<!-- Mark the relevant option(s) with an "x" -->

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Refactoring (no functional changes)
- [ ] CI/CD or infrastructure change
- [ ] Tests only

## Checklist

<!-- Mark completed items with an "x" -->

### Code Quality
- [ ] My code follows the project's style guidelines
- [ ] I have performed a self-review of my code
- [ ] I have commented my code in hard-to-understand areas
- [ ] My changes generate no new warnings

### Testing
- [ ] I have added tests that prove my fix/feature works
- [ ] New and existing unit tests pass locally (`go test -tags "wal,nats" -v -race ./...`)
- [ ] E2E tests pass if applicable (`cd web && npm run test:e2e`)

### Documentation
- [ ] I have updated documentation if needed
- [ ] I have updated the CHANGELOG.md if this is a user-facing change

### Build & Verification
- [ ] The project builds without errors (`go build -tags "wal,nats" -o cartographus ./cmd/server`)
- [ ] Frontend builds without errors (`cd web && npm run build`)
- [ ] Templates are in sync (`./scripts/sync-templates.sh --check`)

## Screenshots/Recordings

<!-- If applicable, add screenshots or recordings to demonstrate the changes -->

## Additional Notes

<!-- Any additional information reviewers should know -->

---

### Reviewer Checklist

- [ ] Code is readable and follows conventions
- [ ] Tests adequately cover the changes
- [ ] No security concerns introduced
- [ ] Documentation is updated if needed
- [ ] CHANGELOG.md updated for user-facing changes
