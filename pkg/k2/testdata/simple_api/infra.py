import klotho
import klotho.aws as aws

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
api.route_to("/", container)
