# Klotho Python IaC SDK

The Klotho Python IaC SDK is a Python package that provides a pythonic interface to build Klotho applications.
The SDK provides a set of APIs to declare and compose Klotho constructs that when executed by the Klotho CLI,
will provision and configure the necessary cloud resources to run the application.

> ⚠️ **Important**
>
> Klotho 2 is in pre-alpha status.
>
> - Some features may be unstable or incomplete.
> - Secrets are currently stored in plaintext.
>
> This SDK is intended for testing and experimentation only and is not suitable for production use.

## Installation

Install the Klotho Python Language SDK using Pipenv:

```bash
pipenv install klotho
```

> ⚠️ **Important**
>
> Other Python package managers are not yet supported by Klotho.
>
> Pipenv is required by the Klotho CLI.

## Usage

The following is an example of a simple Klotho application that creates a Container exposed via an API built using the
Klotho Python Language SDK:

```python
import klotho
import klotho.aws as aws

# Create a Klotho application
app = klotho.Application(
    name="my-app",
    project="my-project",
    environment="dev",
    default_region="us-east-1",
)

# Create a Postgres database
postgres = aws.Postgres(
    "my-postgres",
    username="admin",
    password=os.getenv("DB_PASSWORD"),
    database_name="mydb",
)

# Create a Container resource that binds to the Postgres database
container = aws.Container("my-container", dockerfile="Dockerfile", bindings=[postgres])

# Expose the container via an API
api = aws.API("my-api")
api.route("/my-container", to="container")

```

## Supported Constructs

### AWS

#### Container

The `Container` construct represents a containerized application that will be deployed as an ECS service to AWS Fargate.

**Supported Bindings:**

- `klotho.aws.Postgres`
- `klotho.aws.Bucket`

**Example:**

```python
container = klotho.aws.Container("my-container", dockerfile="Dockerfile")
```

#### API

The `API` construct represents an API Gateway (v1) that can be used to expose a containerized application.

**Example:**

```python
api = klotho.aws.API("my-api")
api.route("/my-container", to=my_container)
```

#### Postgres

The `Postgres` construct represents a PostgreSQL database that can be used by other constructs.

**Example:**

```python
postgres = klotho.aws.Postgres("my-postgres", username="admin", password=os.getenv("DB_PASSWORD"), database_name="mydb")
```

#### FastAPI

The `FastAPI` construct represents a FastAPI application that can be deployed as an ECS service to AWS Fargate.

**Supported Bindings:**

- `klotho.aws.Postgres`
- `klotho.aws.Bucket`

**Example:**

```python
fastapi = klotho.aws.FastAPI("my-fastapi", dockerfile="Dockerfile")
```

#### Bucket

The `Bucket` construct represents an S3 bucket that can be used by other constructs.

**Example:**

```python
bucket = klotho.aws.Bucket("my-bucket")
```

## Support

For questions or issues, please contact the Klotho team on [Discord](https://klo.dev/discordurl).
