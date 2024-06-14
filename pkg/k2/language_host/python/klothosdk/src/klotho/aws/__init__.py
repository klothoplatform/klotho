from .provider import AwsProvider
from ..runtime import instance as runtime
from .bucket import Bucket
from .container import Container

runtime.set_provider("aws", AwsProvider())
