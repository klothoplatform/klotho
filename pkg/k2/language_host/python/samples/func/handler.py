import json


# hello world lambda function
def handler(event, context):
    return {"statusCode": 200, "body": json.dumps("Hello from Lambda!")}
