import klotho
import klotho.aws as aws

app = klotho.Application(
    "my-app",
    project="test_container",
    environment="default",
    default_region="us-west-2",
)

dynamodb = aws.DynamoDB(
    "my-dynamodb",
    attributes=[
        {"Name": "id", "Type": "S"},       # Partition key (Primary Key)
        {"Name": "data", "Type": "S"},     # Sort key (Range Key)
        {"Name": "status", "Type": "S"},   # Attribute for the GSI
        {"Name": "timestamp", "Type": "N"} # Attribute for the LSI
    ],
    hash_key="id",
    range_key="data",  # Range key for the primary key

    # Define a Global Secondary Index (GSI)
    global_secondary_indexes=[
        {
            "name": "StatusIndex",
            "hash_key": "status",                  # Partition key for the GSI
            "projection_type": "ALL"               # Project all attributes
        }
    ],

    # Define a Local Secondary Index (LSI)
    local_secondary_indexes=[
        {
            "name": "TimestampIndex",
            "range_key": "timestamp",              # Sort key for the LSI
            "projection_type": "ALL"               # Project all attributes
        }
    ],
)