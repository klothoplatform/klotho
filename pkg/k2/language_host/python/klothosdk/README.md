# Klotho Pyhon Language SDK

The Klotho Python Language SDK is a Python package that provides a pythonic interface to build Klotho applications.
The SDK provides a set of APIs to declare and compose Klotho constructs that when executed by the Klotho CLI,
will provision and configure the necessary cloud resources to run the application.

## Installation
Install the Klotho Python Language SDK using Pipenv:

```bash
pipenv install klotho
```
> [!IMPORTANT]
> Other Python package managers are not yet supported by Klotho.
> 
> Pipenv is required by the Klotho CLI.

## Usage
The following is an example of a simple Klotho application that creates a Container exposed via an API built using the Klotho Python Language SDK:
```python

import klotho
import klotho.aws as aws

app = klotho.Application(
    name="my-app",
    project="my-project",
)

# Create a Container resource
container = aws.Container("my-container", dockerfile="Dockerfile")

# Expose the container via an API
api = aws.API("my-api")
api.route_to(method="ANY", path="/my-container", dest="container")
```
