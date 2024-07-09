from klotho.aws.api import Api
from klotho.aws.bucket import Bucket
from klotho.aws.container import Container
from klotho.aws.postgres import Postgres
from klotho.aws.fastapi import FastAPI

from klotho.aws.provider import AwsProvider
from klotho.runtime import instance as runtime

runtime.set_provider("aws", AwsProvider())
