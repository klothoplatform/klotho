from .provider import AwsProvider
from klotho.runtime import instance as runtime
from .bucket import Bucket
from .container import Container

runtime.set_provider("aws", AwsProvider())
