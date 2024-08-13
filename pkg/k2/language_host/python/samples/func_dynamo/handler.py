import json
import boto3
import os
from boto3.dynamodb.conditions import Key

# Initialize the DynamoDB client
dynamodb = boto3.resource('dynamodb')
table_name = os.getenv("MY_DYNAMODB_TABLE_NAME")
table = dynamodb.Table(table_name)

# Lambda handler function
def handler(event, context):
    # Example event structure: {"action": "write", "id": "123", "data": "Hello, World!"}
    action = event.get('action')
    id = event.get('id')
    data = event.get('data')
    
    if action == "write" and id and data:
        # Write data to DynamoDB
        table.put_item(Item={"id": id, "data": data})
        return {"statusCode": 200, "body": json.dumps(f"Data written with id: {id}")}
    
    elif action == "read" and id:
        # Query data from DynamoDB based on just id
        response = table.query(
            KeyConditionExpression=Key('id').eq(id)
        )
        items = response.get('Items', [])
        
        if items:
            return {"statusCode": 200, "body": json.dumps(items)}
        else:
            return {"statusCode": 404, "body": json.dumps(f"No items found with id: {id}")}
    
    else:
        return {"statusCode": 400, "body": json.dumps("Invalid action or missing parameters")}
