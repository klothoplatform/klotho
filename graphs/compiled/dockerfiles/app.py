import json

def handler(event, context):
    return {
        'statusCode': 200,
        # 'body': json.dumps({'input': event, 'message': 'Hello from Lambda!'})
        'body': json.dumps({'message': 'Hello from Lambda!'})
    }