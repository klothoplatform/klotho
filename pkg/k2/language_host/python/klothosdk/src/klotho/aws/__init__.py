from klotho.runtime import instance as runtime
from klotho.aws.provider import AwsProvider
from klotho.aws.bucket import Bucket
from klotho.aws.container import Container

runtime.set_provider("aws", AwsProvider())
