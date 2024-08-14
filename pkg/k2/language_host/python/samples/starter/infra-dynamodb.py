import os
from pathlib import Path

import klotho
import klotho.aws as aws

# Create the Application instance
app = klotho.Application(
    "my-dynamo-app",
    project=os.getenv("PROJECT_NAME", "my-project"),
    environment=os.getenv("KLOTHO_ENVIRONMENT", "default"),
    default_region=os.getenv("AWS_REGION", "us-west-2"),  
)

dir = Path(__file__).parent.absolute()

dynamodb = aws.DynamoDB("my-dynamodb", attributes=[{"Name": "id", "Type": "S"}], hash_key="id")