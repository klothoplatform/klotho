import os
from pathlib import Path

import klotho
import klotho.aws as aws

# Create the Application instance
app = klotho.Application(
    "basic",
    project=os.getenv("PROJECT_NAME", "starter"),
)

dir = Path(__file__).parent.absolute()

container = aws.Container(
    "my-container",
    dockerfile=str(dir / "Dockerfile"),
    context=str(dir),
)
