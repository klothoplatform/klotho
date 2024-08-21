import os

import klotho
import klotho.aws as aws

app = klotho.Application(
    "my-app",
    project=os.getenv("PROJECT_NAME", "starter.binding"),
)

bucket = aws.Bucket("my-bucket", force_destroy=True)

container = aws.Container(
    "my-container", dockerfile="Dockerfile", bindings=[bucket.use_read_only()]
)
