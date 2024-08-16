<h1 align="center">
  <br>
  <a href="https://klo.dev"><img src="https://user-images.githubusercontent.com/69910109/209406610-c35afa17-7aff-4d44-921c-078d174d30f0.png" width="300"></a>
  <br>
  Terraform/CDK alternative designed for developers
  <br>
</h1>

[![test badge](https://github.com/klothoplatform/klotho/actions/workflows/test.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/test.yaml)
[![formatting badge](https://github.com/klothoplatform/klotho/actions/workflows/prettier.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/prettier.yaml)
[![linter badge](https://github.com/klothoplatform/klotho/actions/workflows/lint.yaml/badge.svg)](https://github.com/klothoplatform/klotho/actions/workflows/lint.yaml)
[![govulncheck](https://github.com/CloudCompilers/klotho/actions/workflows/govulncheck.yaml/badge.svg)](https://github.com/CloudCompilers/klotho/actions/workflows/govulncheck.yaml)

[![Go Coverage](https://github.com/klothoplatform/klotho/wiki/coverage.svg)](https://raw.githack.com/wiki/klothoplatform/klotho/coverage.html)
[![Latest Release](https://img.shields.io/github/v/release/klothoplatform/klotho)](https://github.com/klothoplatform/klotho/releases)
[![License](https://img.shields.io/github/license/klothoplatform/klotho)](https://github.com/klothoplatform/klotho/blob/main/LICENSE)

## What is Klotho?
Klotho is a developer-centric cloud infra-as-code deployment tool with high level constructs. It lets you think in terms of containers, functions, APIs and databases and combining them freely.

### Example Klotho infra.py
<details>
<summary>infra.py</summary>

```python
import os
from pathlib import Path

import klotho
import klotho.aws as aws

# Create the Application instance
app = klotho.Application(
    "my-sample-app",
    project="my-project",
    environment="default",
    default_region="us-west-2",  
)

dir = Path(__file__).parent.absolute()

# Create a dynamodb instance with 2 indexed attributes
dynamodb = aws.DynamoDB(
    "my-dynamodb",
    attributes=[
        {"Name": "id", "Type": "S"},    
        {"Name": "data", "Type": "S"},  
    ],
    hash_key="id",
    range_key="data"
)

# Create a lambda function that reads in code and deploys it as a zip file
my_function = aws.Function(
    "my-function",
    handler="handler.handler",
    runtime="python3.12",
    code=str(dir),
)

# Bind the dynamodb instance to the lambda function
my_function.bind(dynamodb)

# Create an ECS container
my_container = aws.Container(
    "my-container",
    dockerfile=str(dir / "container" / "Dockerfile"),
    context=str(dir),
)

# Create a Postgres instance with plain text password
my_postgres = aws.Postgres(
    "my-postgres",
    username="admin",
    password="password123!",
    database="mydb",
)

# Bind the postgres instance to the container
my_container.bind(my_postgres)

# Create an API Gateway instance
api = aws.Api("my-api")

# Bind the lambda function to the API Gateway on the /function route
api.route(
  routes: [
    RouteArgs(path="/function", method="ANY")
  ], my_function
)

# Bind the container to the API Gateway on the /container route
api.route(
  routes: [
    RouteArgs(path="/container", method="ANY")
  ], my_container
)
```
</details>

## Getting Started
To get started with Klotho, visit our [documentation](https://klo.dev/docs-k2/) and follow the guides to quickly set up your environment.

## Example Projects
Check out some [example projects](https://github.com/klothoplatform/k2-sample-apps) built using Klotho.

## Community and Support
Join our community of developers and get involved in shaping the future of Klotho:

[![Discord](https://img.shields.io/badge/Klotho-%237289DA.svg?style=for-the-badge&logo=discord&logoColor=white)](https://klo.dev/discordurl)

## Contributing
We welcome contributions from the community. Check out our [contributing guide](https://github.com/klothoplatform/klotho/blob/main/CONTRIBUTING.md) to learn how to get involved in Klotho’s development.

## License
Klotho is licensed under the Apache 2.0 License. See the [LICENSE](https://github.com/klothoplatform/klotho/blob/main/LICENSE) file for more details.
