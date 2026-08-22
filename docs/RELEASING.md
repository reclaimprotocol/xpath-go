# Releasing xpath-go

This guide is for maintainers preparing a tagged release. The GitHub Actions
release job runs only for tags matching `v*` and uses GoReleaser to publish the
GitHub release.

## Prepare the release

1. Start from a clean, up-to-date `main` checkout. Do not include unrelated
   local changes in a release.
2. Choose the version according to [Semantic Versioning](https://semver.org/).
3. Update the user-facing version and release notes:
   - `version.go` (`Version`);
   - `tests/package.json` and its lockfile;
   - `CHANGELOG.md`, using the existing Keep a Changelog structure; and
   - compatibility or API documentation if the public contract changed.
4. Run the pre-merge suite from the repository root:

   ```sh
   make test-all
   ```

   Run `make test-fuzz` and `make test-bench` when the release includes parser,
   evaluator, or performance-sensitive changes. Record benchmark context rather
   than treating one machine’s result as a release gate.
5. Review the exact diff, ensure the version is consistent across release
   materials, and merge the release-preparation pull request.

## Tag and publish

From the reviewed commit on `main`, create and push an annotated tag:

```sh
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

The `release` job requires the quality, test matrix, race, scaling,
compatibility, and cross-build jobs to pass. It has the least additional
permission needed to create the GitHub release (`contents: write`). Monitor the
workflow and verify the generated release notes, source archive, checksums, and
artifacts before announcing the release.

## Correcting a release

Do not move or reuse a published version tag. If a release needs a correction,
prepare and publish the next patch version with a clear changelog entry. If a
security issue is involved, follow [SECURITY.md](../SECURITY.md) before
disclosing details.
