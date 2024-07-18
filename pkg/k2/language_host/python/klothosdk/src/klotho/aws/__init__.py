from klotho.runtime import instance as runtime
from klotho.aws.provider import AwsProvider
from klotho.aws.bucket import Bucket
from klotho.aws.container import Container
from klotho.aws.postgres import Postgres
from klotho.aws.fastapi import FastAPI

runtime.set_provider("aws", AwsProvider())
