

<h1 align="center">
  <br>
  <a href="https://klo.dev"><img src="https://user-images.githubusercontent.com/69910109/209406610-c35afa17-7aff-4d44-921c-078d174d30f0.png" width="300"></a>
  <br>
  develop for local, deploy for the cloud
  <br>
</h1>

[![test badge](https://github.com/klothoplatform/klotho/actions/workflows/test.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/test.yaml)
[![formatting badge](https://github.com/klothoplatform/klotho/actions/workflows/prettier.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/prettier.yaml)
[![linter badge](https://github.com/klothoplatform/klotho/actions/workflows/lint.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/lint.yaml)
[![staticcheck badge](https://github.com/klothoplatform/klotho/actions/workflows/staticcheck.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/staticcheck.yaml)
[![release badge](https://github.com/klothoplatform/klotho/actions/workflows/release.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/release.yaml)

Klotho is an open source tool that transforms plain code into cloud native code.

Klotho allows you to quickly and reliably add cloud functionality to your application with minimal modification to your code. In most cases, this is just a handful of klotho annotations.

 It adds 3 main in-code cloud capabilities:
- **`expose`** web APIs to the Internet
- **`persist`** multi-modal data into different types of databases
- **`static_unit`** package static assets and upload into a CDN for distribution

## Table of Contents

* [Why?](#why)
* [Adaptive Architectures](#adaptive-architectures)
* [Infrastructure-from-Code](#infrastructure-from-code)
* [Installation](#installation)
* [Getting Started](#getting-started)
* [Example usage](#example-usage)
* [Additional Resources](#additional-resources)
* [Language Support](#language-support)
* [Cloud Providers](#cloud-providers)
* [Developing](#developing)

## Why?

Klotho is designed to absorb the complexity of building cloud applications, enabling everyone in large-scale organizations and teams to hobbyist developers to write and operate cloud applications at a fraction of the effort. 

Its design principles are an outcome of industry collaborations focused on mid-sized companies and fast growing startups.

## Adaptive Architectures

Klotho builds on a new architecture called [Adaptive Architecutes](https://www.youtube.com/watch?v=GHt3FAEDfII&t=40392s). 

![image](https://user-images.githubusercontent.com/69910109/209458345-261875db-7168-4570-86ac-f43fe37f78c6.png)

It's a superset of monoliths, microservices and serverless architectures, combining their benefits like a stellar developer experience, immediate productivity, scalability and autonomy of development and deployment as well as a spectrum of NoOps to FullOps. It also introduces a host of [new capabilities](https://klo.dev/) that have been out of reach do to their implementation complexity. 

## Infrastructure-from-Code

Klotho is part of a new generation of cloud tools that implements Infrastructure-from-Code (IfC), a process to automatically create, configure and manage cloud resources from the existing software application's source code without having describe it explicitly. 

By annotating the clients, SDKs or language constructs used in the code with Klotho capabilities, they are automatically created, updated and wired into the application. 

<p align="center" style="font-size: 11px">
  <br>
  <a href="https://klo.dev"><img src="https://user-images.githubusercontent.com/69910109/209459034-8478468a-119e-4feb-a963-7201cfc9e360.png"></a>
  <br>
  <span>Exposing a Python FastAPI to the internet with the Klotho <code>klotho::expose</code> capability. View for <a href="">NodeJS</a>, <a href="">Go</a> </span>
  <br>
</p>

<p align="center" style="font-size: 11px">
  <br>
  <a href="https://klo.dev"><img src="https://user-images.githubusercontent.com/69910109/209459591-5a4cd026-42ec-4a30-8d7a-9047d3760989.png"></a>
  <br>
  <span>Persisting Redis and TypeORM instances in NodeJS with the the Klotho <code>klotho::persist</code> capability. View for <a href="">Python</a>, Go (soon) </span>
  <br>
</p>

Klotho ensures that developers/operators are able to select and adapt the underlying technologies even after their initial setup.

## Installation

To install the latest Klotho release, run the following (see [full installation instructions](https://klo.dev/docs/download-klotho) for additional installation options):

Mac:

```sh
brew install klothoplatform/tap/klotho
```

Linux/WSL2:
```sh
curl -fsSL "https://github.com/klothoplatform/klotho/releases/latest/download/klotho_linux_amd64" -o klotho
chmod +x klotho
  ```


## Getting Started

The quickest way to get started is with the getting started tutorial for [Javascript/Typescript](https://klo.dev/docs/tutorials/your_first_klotho_app), [Python](https://klo.dev/docs/tutorials/your_first_klotho_app_python) and Go (soon). 

## Example usage
### Clone the sample app
Clone our sample apps git repo and install the npm packages for the [js-my-first-app](https://github.com/KlothoPlatform/sample-apps/tree/main/js-my-first-app) app:

```sh
git clone https://github.com/KlothoPlatform/sample-apps.git
cd sample-apps/js-my-first-app
npm install
```

### Logging in
First log in to Klotho. This shares telemetry data for compiler improvements:
```sh
klotho --login <email>
```

### Compile with Klotho
Now compile the application for AWS by running `klotho` and passing `--provider aws` as an argument on the command line.

```sh
klotho . --app my-first-app --provider aws
```

Will result in:
```sh
██╗  ██╗██╗      ██████╗ ████████╗██╗  ██╗ ██████╗
██║ ██╔╝██║     ██╔═══██╗╚══██╔══╝██║  ██║██╔═══██╗
█████╔╝ ██║     ██║   ██║   ██║   ███████║██║   ██║
██╔═██╗ ██║     ██║   ██║   ██║   ██╔══██║██║   ██║
██║  ██╗███████╗╚██████╔╝   ██║   ██║  ██║╚██████╔╝
╚═╝  ╚═╝╚══════╝ ╚═════╝    ╚═╝   ╚═╝  ╚═╝ ╚═════╝

Adding resource input_file_dependencies:
Adding resource exec_unit:main
Found 2 route(s) on server 'app'
Adding resource gateway:pet-api
Adding resource persist_kv:petsByOwner
Adding resource topology:my-first-app
Adding resource infra_as_code:Pulumi (AWS)
```

The cloud version of the application is saved to the `./compiled` directory, and has everything you need to deploy, run and operate the application.

### Examine the result
As part of the compilation, Klotho generates a high-level topology diagram showing the cloud resources that will be used in your application's cloud deployment and their relationships.

Open `./compiled/my-first-app.png` to view the application's topology diagram:

<div align="center">
 <img src="https://klo.dev/docs/assets/images/creating_rest_api_topo-d5c9e11b53e45403d374e02e3b28a34d.png" width="300">
</div>

We can see here that Klotho has defined the following AWS topology:

- **main** ([Lambda](https://aws.amazon.com/lambda/)) - The main Lambda function serves the Express app defined in js-my-first-app using a Lambda-compatible interface.
- **pet-api** ([API Gateway](https://aws.amazon.com/api-gateway/)) - The pet-api API gateway is used to expose the Express routes defined in the main Lambda function.
- **petsByOwner** ([DynamoDB Table](https://aws.amazon.com/dynamodb/)) - The petsByOwner DynamoDB table is used by the main Lambda function to store the relationships between pets and their owners.

[Continue reading the tutorial](https://klo.dev/docs/tutorials/your_first_klotho_app)

## Additional Resources

Here are some links to additional resources:

- [Documentation](https://klo.dev/docs)
- Case Studies: [Amihan Entertainment](https://klo.dev/case-study-amihan-entertainment/), Remedy
- [Blog](https://klo.dev/blog/)
- Podcasts: [How to Reduce Cloud Development Complexity](https://www.devopsparadox.com/episodes/how-to-reduce-cloud-development-complexity-169/)

## Language Support

### Supported
These languages support the majority of capabilities and a wide variety of code styles.

![Languages](https://skillicons.dev/icons?i=ts,js,python)

### Early Access
These languages support only a minority of capabilities and/or small subset of code styles.

![Languages](https://skillicons.dev/icons?i=go)

### In-development
These languages are not yet supported but are in design and development

![Languages](https://skillicons.dev/icons?i=cs,java)

## Cloud Providers

### Supported
These providers support the majority of capabilities and languages.

![Cloud providers](https://skillicons.dev/icons?i=aws)

### In-development
These providers are not yet supported but are in design and development

![Cloud providers](https://skillicons.dev/icons?i=gcp,azure)

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
