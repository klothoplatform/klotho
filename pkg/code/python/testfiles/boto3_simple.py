import boto3

s3 = boto3.resource('s3')

def using_resource():
    bucket = s3.Bucket('my-bucket')
    myobj = bucket.Object('my-object')
