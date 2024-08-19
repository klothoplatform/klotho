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

dynamodb = aws.DynamoDB(
    "my-dynamodb",
    attributes=[
        {"Name": "id", "Type": "S"},    # Partition key (Primary Key)
        {"Name": "data", "Type": "S"},  # Attribute for indexing
    ],
    hash_key="id",

    # Define a Global Secondary Index (GSI)
    global_secondary_indexes=[
        {
            "Name": "DataIndex",
            "HashKey": "data",                  # Partition key for the GSI
            "ProjectionType": "ALL"             # Project all attributes
        }
    ],
)

my_function = aws.Function(
    "my-function",
    handler="handler.handler",
    runtime="python3.12",
    code=str(dir),
)

my_function.bind(dynamodb)