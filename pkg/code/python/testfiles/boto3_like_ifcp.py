import os
import boto3

def create_s3_resource():
    if os.getenv("ARCHITECTURE_BUCKET_NAME", None) is None:
        return boto3.resource(
            "s3",
            endpoint_url="http://localhost:9000",
            aws_access_key_id="minio",
            aws_secret_access_key="minio123",
        )
    else:
        return boto3.resource("s3")

def get_architecture_storage():
    bucket_name = os.getenv("ARCHITECTURE_BUCKET_NAME", "ifcp-architecture-storage")
    s3_resource = create_s3_resource()
    return ArchitectureStorage(bucket=s3_resource.Bucket(bucket_name))  # type: ignore


def get_binary_storage():
    bucket_name = os.getenv("BINARY_BUCKET_NAME", "ifcp-binary-storage")
    s3_resource = create_s3_resource()
    return BinaryStorage(bucket=s3_resource.Bucket(bucket_name))  # type: ignore
