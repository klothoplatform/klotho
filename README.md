# Welcome to [Klotho](https://klo.dev)
[![test badge](https://github.com/klothoplatform/klotho/actions/workflows/test.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/test.yaml)
[![formatting badge](https://github.com/klothoplatform/klotho/actions/workflows/prettier.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/prettier.yaml)
[![linter badge](https://github.com/klothoplatform/klotho/actions/workflows/lint.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/lint.yaml)
[![staticcheck badge](https://github.com/klothoplatform/klotho/actions/workflows/staticcheck.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/staticcheck.yaml)
[![integ tests badge](https://github.com/klothoplatform/klotho/actions/workflows/integtest.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/integtest.yaml)
[![release badge](https://github.com/klothoplatform/klotho/actions/workflows/release.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/release.yaml)

## Introduction

Klotho is a new development model that enables anyone, from large-scale organizations and teams to hobby developers, to write and operate cloud applications at a fraction of the effort. Klotho provides all the benefits and capabilities of monolith, microservice, and serverless architectures while making it simple to create, combine, and migrate between these architectures without modifying your code. With Klotho, you can create cloud applications, support event-driven workloads and architectures, leverage machine-learning models, and expose web APIs. Support demanding performance requirements like compute, latency, and reliability while optimizing for cost.

Our offering is centered around our design principles:

- maintain benefits from existing architectures
- keep existing tools and programming languages usable
- integrate with an ecosystem instead of trying to replace it
- ensure user code is recognizable, debuggable, and patchable–even in production

We’re happy to announce our closed beta, [available now](https://l.klo.dev/signup/closed-beta).

## Additional Resources

Here are some links to additional resources:

- [Documentation](https://klo.dev/docs)
- Case Studies: [Amihan Entertainment](https://klo.dev/case-study-amihan-entertainment/), Remedy
- [Blog](https://klo.dev/blog/)
- Podcasts: [How to Reduce Cloud Development Complexity](https://www.devopsparadox.com/episodes/how-to-reduce-cloud-development-complexity-169/)

## Target Languages

![Languages](https://skillicons.dev/icons?i=ts,js,python,go,cs,java)

## Target Cloud Providers

![Cloud providers](https://skillicons.dev/icons?i=aws,gcp,azure)

## Coming soon

Our next phase, the open beta, will mark a critical milestone in our development as we open-source Klotho’s core. We’ll be introducing support for additional languages and cloud providers, along with several experimental VSCode extensions for syntax highlighting and annotation snippets.

We look forward to sharing more details in the coming weeks. Please reach out with any feedback or inquiries!

[![Twitter](https://img.shields.io/badge/Twitter-%231DA1F2.svg?style=for-the-badge&logo=Twitter&logoColor=white)](https://twitter.com/GetKlotho) [![LinkedIn](https://img.shields.io/badge/linkedin-%230077B5.svg?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/company/klothoplatform/)
[![Discord](https://img.shields.io/badge/%3CServer%3E-%237289DA.svg?style=for-the-badge&logo=discord&logoColor=white)](https://discord.gg/4z2jwRvnyM)

## Developing
* build: `go build ./...`
* test: `go test ./...`
* run without separate build: `go run ./cmd/klotho`
* to run CI checks on `git push`:
```
git config --local core.hooksPath .githooks/
```
