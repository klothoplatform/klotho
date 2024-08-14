import klotho
import klotho.aws as aws

app = klotho.Application(
    "my-app",
    project="test_container",
    environment="default",
    default_region="us-west-2",
)

bucket = aws.Bucket("my-bucket", force_destroy=True)
container = aws.Container(
    "my-container",
    dockerfile="/Dockerfile",
    context="/",
    bindings=[bucket.use_read_only()],
)
