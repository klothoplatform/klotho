import os

import klotho
import klotho.aws as aws

# Create the Application instance
app = klotho.Application(
    "my-app",
    project=os.getenv(
        "PROJECT_NAME", "my-project"
    ),  # Default to 'my-project' or the environment variable value
    environment=os.getenv("KLOTHO_ENVIRONMENT", "default"),
    # Default to 'default' or the environment variable value
    default_region=os.getenv(
        "AWS_REGION", "us-west-2"
    ),  # Default to 'us-east-1' or the environment variable value
)

container = aws.Container("my-container", dockerfile="Dockerfile")

api = aws.Api("my-api")
api.route_to("GET", "/hello", container)
