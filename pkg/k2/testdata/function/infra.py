from pathlib import Path

import klotho
import klotho.aws as aws
from klotho.aws.api import RouteArgs

klotho.Application(
    "my-app",
    project=Path(__file__).parent.name,
    environment="default",
    default_region="us-west-2",
)

bucket = aws.Bucket("my-bucket", force_destroy=True)

docker_func = aws.Function("docker-func", dockerfile="/Dockerfile", docker_context="/")
zip_func = aws.Function(
    "zip-func",
    code="/",
    handler="handler.handler",
    runtime="python3.12",
    bindings=[bucket.use_read_only()],
)

api = aws.Api("my-api")
api.route([RouteArgs(path="/")], destination=docker_func)
