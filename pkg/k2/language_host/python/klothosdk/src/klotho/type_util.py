from typing import Any, Optional, TypeVar, Type

from klotho import Output
from klotho.construct import Construct


def set_field(self, key: str, value: Any) -> None:
    """
    Set a value in the instance's dictionary.

    :param key: The key to set.
    :param value: The value to set.
    """
    self.__dict__[key] = value


def get_field(self, key: str) -> Optional[Any]:
    """
    Get a value from the instance's dictionary.

    :param key: The key to get.
    :return: The value, if it exists.
    """
    return self.__dict__.get(key)


T = TypeVar("T")


def get_output[T](self: Construct, path: str, output_type: Type[T] = any) -> Output[T]:
    """
    Get an output of the specified type from the resource at the specified path.

    :param self:
    :param output_type:  The type of the output.
    :param path: The path to the output.
    :return: The output.
    """

    output_urn = str(self.urn.with_output(path))

    return Output[output_type](
        id=output_urn,
        depends_on={output_urn},
    )
