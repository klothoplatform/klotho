import os

import klotho
import klotho.aws as aws
from klotho.aws.bucket import BucketArgs
from klotho.construct import ConstructOptions

# Create the Application instance
app = klotho.Application(
    "my-app",
    project=os.getenv(
        "PROJECT_NAME", "my-project"
    ),  # Default to 'my-project' or the environment variable value
    environment=os.getenv("KLOTHO_ENVIRONMENT", "default"),
    # Default to 'default' or the environment variable value
    default_region=os.getenv(
        "AWS_REGION", "us-east-1"
    ),  # Default to 'us-east-1' or the environment variable value
)

# Create a Container resource
# container = aws.Container('my-container', image="my-image:latest")

# Create a Bucket resource
bucket = aws.Bucket(
    "my-bucket",
    BucketArgs(
        index_document="index.html",
        sse_algorithm="AES256",
        force_destroy=True,
    ),
    opts=ConstructOptions(),
)
