from typing import override, Type

from klotho.aws.network import Network
from klotho.provider import Provider, T
from klotho.runtime import instance as runtime


class AwsProvider(Provider):

    @override
    def get_default_construct(self, construct_type: Type[T]) -> T:
        if construct_type == Network:
            default = runtime.constructs.get("default-network")
            return default if default is not None else Network("default-network")
        raise ValueError(f"No default found for construct type: {construct_type}")
