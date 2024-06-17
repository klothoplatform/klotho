from typing import TypeVar, Generic, Union, Mapping, Any, TYPE_CHECKING, Callable
from uuid import uuid4

from klotho.runtime import instance as runtime

if TYPE_CHECKING:
    pass

T = TypeVar('T')
T_co = TypeVar("T_co", covariant=True)

Input = Union[T, "Output[T]"]
Inputs = Mapping[str, Input[Any]]
InputType = Union[T, Mapping[str, Any]]


class Output(Generic[T_co]):
    def __init__(
            self,
            depends_on: set[str],
            id: str = None,
            value: T_co | Input[T_co] | None = None,
            callback: Callable[[T_co], T] = None,
    ):
        if id is not None:
            if id in runtime.output_references:
                raise ValueError(f"Output with id {id} already exists")
            self.id = id
        else:
            self.id = str(uuid4())
        self.depends_on = set(depends_on)

        if isinstance(value, Output) and value:
            self._value = value.value
            self._is_resolved = value.is_resolved
        self._value = value
        self._is_resolved = False
        self.callback = callback

        runtime.output_references[self.id] = self

    @property
    def is_resolved(self) -> bool:
        return self._is_resolved

    @property
    def value(self) -> T_co:
        if not self._is_resolved:
            raise ValueError("Output value is not resolved")
        return self._value

    def __str__(self):
        return f"Output({self.id}={self._value})"

    def apply(self, callback: Callable[[T_co], T]) -> "Output[T]":
        def run(value: T_co) -> T_co:
            if self._is_resolved:
                return self._value
            self._value = callback(value)
            self._is_resolved = True
            return self._value

        return Output(self.depends_on, self._value, run)

    def resolve(self, resolved_deps):
        if self._is_resolved:
            return
        self._value = self.callback(*resolved_deps) if self.callback else resolved_deps[0]
        self._is_resolved = True
        runtime.outputs[self.id] = self._value

    @staticmethod
    def all(outputs: list["Output[T_co]"], callback: Callable[..., T_co] = None) -> "Output[T_co]":
        def run(*values: T_co) -> T_co:
            return callback(*values)

        return Output({
            *[output.id for output in outputs],
            *[dep for output in outputs for dep in output.depends_on]
        }
            , None, None, run)
