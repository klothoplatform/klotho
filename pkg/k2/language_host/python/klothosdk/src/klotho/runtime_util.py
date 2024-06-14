from typing import TypeVar, Type

from klotho.construct import Construct
from klotho.runtime import instance as runtime

Tc = TypeVar('Tc', bound=Construct)


def get_default_construct(provider: str, construct_type: Type[Tc]) -> Tc:
    provider = runtime.providers.get(provider)
    if provider is None:
        raise ValueError(f"Provider {provider} not found")
    return provider.get_default_construct(construct_type)
