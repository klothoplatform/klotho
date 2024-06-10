from abc import ABC
from typing import Type, TypeVar

T = TypeVar('T', bound='Construct')


class Provider(ABC):

    def get_default_construct(self, construct_type: Type[T]) -> T:
        pass
