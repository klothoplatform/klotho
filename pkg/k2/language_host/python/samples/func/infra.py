import os
from pathlib import Path

import klotho
import klotho.aws as aws

# Create the Application instance
app = klotho.Application(
    "app",
    project=os.getenv("PROJECT_NAME", "func"),
)

dir = Path(__file__).parent.absolute()

my_function = aws.Function(
    "my-function",
    handler="handler.handler",
    runtime="python3.12",
    code=str(dir),
)
