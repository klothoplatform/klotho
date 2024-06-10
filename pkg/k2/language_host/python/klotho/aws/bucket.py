from typing import Optional, overload

from klotho.output import Input, Output
from klotho.construct import ConstructOptions, get_construct_args_opts, Construct
from klotho.type_util import set, get, get_output


class BucketArgs:
    def __init__(self,
                 index_document: Optional[Input[str]] = None,
                 sse_algorithm: Optional[Input[str]] = None,
                 force_destroy: Optional[Input[bool]] = None):
        if index_document is not None:
            set(self, "index_document", index_document)
        if sse_algorithm is not None:
            set(self, "sse_algorithm", sse_algorithm)
        if force_destroy is not None:
            set(self, "force_destroy", force_destroy)

    @property
    def index_document(self) -> Optional[Input[str]]:
        return get(self, "index_document")

    @index_document.setter
    def index_document(self, value: Optional[Input[str]]) -> None:
        set(self, "index_document", value)

    @property
    def sse_algorithm(self) -> Optional[Input[str]]:
        return get(self, "sse_algorithm")

    @sse_algorithm.setter
    def sse_algorithm(self, value: Optional[Input[str]]) -> None:
        set(self, "sse_algorithm", value)

    @property
    def force_destroy(self) -> Optional[Input[bool]]:
        return get(self, "force_destroy")

    @force_destroy.setter
    def force_destroy(self, value: Optional[Input[bool]]) -> None:
        set(self, "force_destroy", value)


class Bucket(Construct):

    @overload
    def __init__(self, name: str, args: BucketArgs, opts: Optional[ConstructOptions] = None):
        ...

    @overload
    def __init__(self, name,
                 index_document: Optional[Input[str]] = None,
                 sse_algorithm: Optional[Input[str]] = None,
                 force_destroy: Optional[Input[bool]] = None,
                 opts: Optional[ConstructOptions] = None):
        ...

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

    @property
    def arn(self) -> Output[str]:
        return get_output(self, path="Arn", output_type=str)

    @property
    def bucket_name(self) -> Output[str]:
        return get_output(self, path="BucketName", output_type=str)
