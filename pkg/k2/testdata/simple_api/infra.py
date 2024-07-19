import klotho
import klotho.aws as aws
from klotho.aws.api import RouteArgs

app = klotho.Application(
    "my-app",
    project="test_container",
    environment="default",
    default_region="us-west-2",
)

container = aws.Container(
    "my-container",
    dockerfile="/Dockerfile",
    context="/",
)

api = aws.Api("my-api")
api.route([RouteArgs(path="/")], destination=container)
