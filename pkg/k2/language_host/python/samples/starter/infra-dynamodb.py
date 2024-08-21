import os
from pathlib import Path

import klotho
import klotho.aws as aws

# Create the Application instance
app = klotho.Application(
    "dynamodb",
    project=os.getenv("PROJECT_NAME", "starter"),
)

dir = Path(__file__).parent.absolute()

dynamodb = aws.DynamoDB("my-dynamodb", attributes=[{"Name": "id", "Type": "S"}], hash_key="id")
