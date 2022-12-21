import boto3
import os

payloadBucketPhysicalName = os.getenv("KLOTHO_S3_PREFIX") + "{{.PayloadsBucketName}}"


def open(url: str, **kwargs):
    return s3ContextManager(str(url))


class s3ContextManager(object):
    def __init__(self, file_name: str):
        self.client = boto3.client('s3')
        self.bucket_name = payloadBucketPhysicalName
        self.file_name = file_name

    async def __aenter__(self):
        return FsItem(bucket_name=self.bucket_name, file_name=self.file_name, client=self.client)

    async def __aexit__(self, exc_type, exc_val, traceback):
        pass


class FsItem(object):
    def __init__(self, file_name: str, bucket_name: str, client: boto3.client, **kwargs):
        self.client = client
        self.bucket_name = bucket_name
        self.file_name = file_name

    async def write(self, content: str):
        self.client.put_object(Key=self.file_name, Bucket=self.bucket_name, Body=content)

    async def read(self):
        response = self.client.get_object(Key=self.file_name, Bucket=self.bucket_name)
        return response["Body"]
