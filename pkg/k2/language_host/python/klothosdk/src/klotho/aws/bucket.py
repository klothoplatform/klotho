from typing import Optional, overload, Any

from klotho.construct import (
    ConstructOptions,
    get_construct_args_opts,
    Construct,
    Binding,
)
from klotho.output import Input, Output
from klotho.type_util import set_field, get_field, get_output


class BucketArgs:
    def __init__(
        self,
        index_document: Optional[Input[str]] = None,
        sse_algorithm: Optional[Input[str]] = None,
        force_destroy: Optional[Input[bool]] = None,
    ):
        if index_document is not None:
            set_field(self, "index_document", index_document)
        if sse_algorithm is not None:
            set_field(self, "sse_algorithm", sse_algorithm)
        if force_destroy is not None:
            set_field(self, "force_destroy", force_destroy)

    @property
    def index_document(self) -> Optional[Input[str]]:
        return get_field(self, "index_document")

    @index_document.setter
    def index_document(self, value: Optional[Input[str]]) -> None:
        set_field(self, "index_document", value)

    @property
    def sse_algorithm(self) -> Optional[Input[str]]:
        return get_field(self, "sse_algorithm")

    @sse_algorithm.setter
    def sse_algorithm(self, value: Optional[Input[str]]) -> None:
        set_field(self, "sse_algorithm", value)

    @property
    def force_destroy(self) -> Optional[Input[bool]]:
        return get_field(self, "force_destroy")

    @force_destroy.setter
    def force_destroy(self, value: Optional[Input[bool]]) -> None:
        set_field(self, "force_destroy", value)


class Bucket(Construct):

    @overload
    def __init__(
        self, name: str, args: BucketArgs, opts: Optional[ConstructOptions] = None
    ): ...

    @overload
    def __init__(
        self,
        name,
        index_document: Optional[Input[str]] = None,
        sse_algorithm: Optional[Input[str]] = None,
        force_destroy: Optional[Input[bool]] = None,
        opts: Optional[ConstructOptions] = None,
    ): ...

    def __init__(self, name, *args, **kwargs):
        construct_args, opts = get_construct_args_opts(BucketArgs, *args, **kwargs)
        if construct_args is not None:
            self._internal_init(name, opts, **construct_args.__dict__)
        else:
            self._internal_init(name, *args, **kwargs)

    def _internal_init(
        self,
        name: str,
        opts: Optional[ConstructOptions] = None,
        index_document: Optional[Input[str]] = None,
        sse_algorithm: Optional[Input[str]] = None,
        force_destroy: Optional[Input[bool]] = None,
    ):
        super().__init__(
            name,
            construct_type="klotho.aws.Bucket",
            properties={
                "IndexDocument": index_document,
                "SseAlgorithm": sse_algorithm,
                "ForceDestroy": force_destroy,
            },
            opts=opts,
        )

    # Outputs
    @property
    def arn(self) -> Output[str]:
        return get_output(self, path="Arn", output_type=str)

    @property
    def bucket(self) -> Output[str]:
        return get_output(self, path="Bucket", output_type=str)

    # Bindings
    def use_read_only(self):
        """
        This method is used to create a binding for the bucket construct with read-only permissions.
        :return:
        """
        return Binding(self, inputs={"ReadOnly": True})

    def use_read_write(self):
        """
        This method is used to create a binding for the bucket construct with read-write permissions.
        :return: Binding
        """
        return Binding(self, inputs={"ReadOnly": False})
