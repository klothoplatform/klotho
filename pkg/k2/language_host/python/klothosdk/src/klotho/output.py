from typing import (
    TypeVar,
    Generic,
    Union,
    Mapping,
    Any,
    TYPE_CHECKING,
    Callable,
    Optional,
    overload,
)
from uuid import uuid4

from klotho.runtime import instance as runtime

if TYPE_CHECKING:
    pass

T = TypeVar("T")
K = TypeVar("K")
T_co = TypeVar("T_co", covariant=True)

Input = Union[T, "Output[T]"]
MappingInput = Union[Mapping[str, Input[T]], Input[Mapping[str, T]]]


class Output(Generic[T_co]):
    def __init__(
        self,
        depends_on: Optional[set[str]] = None,
        id: Optional[str] = None,
        value: Optional[T_co | Input[T_co]] = None,
        callback: Optional[Callable[[T_co], T]] = None,
    ):
        if id is not None:
            if id in runtime.output_references:
                raise ValueError(f"Output with id {id} already exists")
            self.id = id
        else:
            self.id = str(uuid4())
        self.depends_on = set(depends_on) if depends_on else set()

        if isinstance(value, Output) and value:
            self._value = value.value
            self._is_resolved = value.is_resolved
        self._value = value
        self._is_resolved = value is not None
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
        self._value = (
            self.callback(*resolved_deps) if self.callback else resolved_deps[0]
        )
        self._is_resolved = True
        runtime.outputs[self.id] = self._value

    @staticmethod
    @overload
    def all(
        outputs: list["Output[T_co]"], callback: Callable[..., T_co] | None = None
    ) -> "Output[T_co]": ...

    @staticmethod
    @overload
    def all(
        outputs: Mapping[K, "Output[T_co]"],
        callback: Callable[[Mapping[K, T_co]], T_co] | None = None,
    ) -> "Output[T_co]": ...

    @staticmethod
    def all(
        outputs: list["Output[T_co]"] | Mapping[Any, "Output[T_co]"],
        callback: (
            Callable[..., T_co] | Callable[[Mapping[Any, T_co]], T_co] | None
        ) = None,
    ) -> "Output[T_co]":
        """
        Creates a new Output that represents the output of applying a callback function to the given outputs.
        Accepts either a list of Output objects or a Mapping of keys to Output objects.
        :param outputs: List of Output objects or Mapping of keys to Output objects
        :param callback: Callback function to apply to the outputs
        :return: The new Output
        """
        if isinstance(outputs, Mapping):

            output_list = []
            output_keys = []
            for key, output in outputs.items():
                output_list.append(output)
                output_keys.append(key)

            def run(*values: T_co) -> T_co:
                resolved_outputs = {
                    key: value for key, value in zip(output_keys, values)
                }
                return callback(resolved_outputs)  # type: ignore

        else:

            def run(*values: T_co) -> T_co:
                return callback(*values)  # type: ignore

            output_list = outputs
            output_keys = None

        return Output(
            depends_on={
                *[output.id for output in output_list],
                *[dep for output in output_list for dep in output.depends_on],
            },
            callback=run,
        )

    @classmethod
    def concat(cls, *args: "Input[str]") -> "Output[str]":
        """
        Concatenates the string representations of all the given inputs.
        :param args: The inputs to concatenate.
        :return: A new Output representing the concatenated string.
        """
        inputs = [
            arg if isinstance(arg, Output) else Output(set(), None, arg) for arg in args
        ]

        def run(*values: str) -> str:
            return "".join(values)

        return cls.all(inputs, run)

    @staticmethod
    def from_mapping(input: Input[Mapping]) -> "Output[Mapping]":
        if isinstance(input, Output):
            return input
        if isinstance(input, Mapping):
            deps = set()
            unresolved_mappings = {}
            resolved_mappings = {}
            for key, value in input.items():
                if isinstance(value, Output):
                    deps.update(value.depends_on)
                    unresolved_mappings[key] = value
                else:
                    resolved_mappings[key] = value

            def callback(resolved_outputs: Mapping) -> Mapping:
                result = {**resolved_mappings, **resolved_outputs}
                return result

            return Output.all(
                unresolved_mappings,
                callback=callback,
            )
        else:
            raise ValueError("Input must be an Output or a Mapping")
