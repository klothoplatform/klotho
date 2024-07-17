
## Creating a release

Releases are created with the [Publish K2 Python Language SDK](.github/workflows/publish-k2-python-language-sdk.yaml) GitHub Actions workflow.
The workflow is executed manually by running the workflow from the GitHub Actions tab in the repository.

When creating a release, you can specify the following inputs:
- `environment`: The environment to release the package to. Options are `test` and `prod`. Default is `test`. The environment refers to which index the package will be published to: TestPyPI or PyPI.
- `version`: The version of the package to release. The version must be a valid [PEP 440](https://pep440.readthedocs.io/en/latest/) version or one of the following values:
    - `a|b|rc|post|dev`: The next alpha, beta, release-candidate, post-release, or development release, respectively. The released version will have the current run number appended to the version.

  If the version is not specified, the workflow will create a development release if the environment is `test` or use the version from `pyproject.toml` if the environment is `prod`.