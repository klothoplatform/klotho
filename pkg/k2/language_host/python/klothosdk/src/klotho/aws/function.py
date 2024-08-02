from typing import Optional, overload, Mapping, Any

from klotho.construct import ConstructOptions, get_construct_args_opts, Construct
from klotho.output import Input, Output, MappingInput
from klotho.type_util import set_field, get_field, get_output


class FunctionArgs:
    def __init__(
        self,
        handler: Optional[Input[str]] = None,
        runtime: Optional[Input[str]] = None,
        timeout: Optional[Input[int]] = None,
        memory_size: Optional[Input[int]] = None,
        environment_variables: Optional[MappingInput[str]] = None,
        code: Optional[Input[str]] = None,
        s3_bucket: Optional[Input[str]] = None,
        s3_key: Optional[Input[str]] = None,
        s3_object_version: Optional[Input[str]] = None,
        image_uri: Optional[Input[str]] = None,
    ):
        if handler is not None:
            set_field(self, "handler", handler)
        if runtime is not None:
            set_field(self, "runtime", runtime)
        if timeout is not None:
            set_field(self, "timeout", timeout)
        if memory_size is not None:
            set_field(self, "memory_size", memory_size)
        if environment_variables is not None:
            set_field(self, "environment_variables", environment_variables)
        if code is not None:
            set_field(self, "code", code)
        if s3_bucket is not None:
            set_field(self, "s3_bucket", s3_bucket)
        if s3_key is not None:
            set_field(self, "s3_key", s3_key)
        if s3_object_version is not None:
            set_field(self, "s3_object_version", s3_object_version)
        if image_uri is not None:
            set_field(self, "image_uri", image_uri)

    @property
    def handler(self) -> Optional[Input[str]]:
        return get_field(self, "handler")

    @handler.setter
    def handler(self, value: Optional[Input[str]]) -> None:
        set_field(self, "handler", value)

    @property
    def runtime(self) -> Optional[Input[str]]:
        return get_field(self, "runtime")

    @runtime.setter
    def runtime(self, value: Optional[Input[str]]) -> None:
        set_field(self, "runtime", value)

    @property
    def timeout(self) -> Optional[Input[int]]:
        return get_field(self, "timeout")

    @timeout.setter
    def timeout(self, value: Optional[Input[int]]) -> None:
        set_field(self, "timeout", value)

    @property
    def memory_size(self) -> Optional[Input[int]]:
        return get_field(self, "memory_size")

    @memory_size.setter
    def memory_size(self, value: Optional[Input[int]]) -> None:
        set_field(self, "memory_size", value)

    @property
    def environment_variables(self) -> Optional[MappingInput[str]]:
        return get_field(self, "environment_variables")

    @environment_variables.setter
    def environment_variables(self, value: Optional[MappingInput[str]]) -> None:
        set_field(self, "environment_variables", value)

    @property
    def code(self) -> Optional[Input[str]]:
        return get_field(self, "code")

    @code.setter
    def code(self, value: Optional[Input[str]]) -> None:
        set_field(self, "code", value)

    @property
    def s3_bucket(self) -> Optional[Input[str]]:
        return get_field(self, "s3_bucket")

    @s3_bucket.setter
    def s3_bucket(self, value: Optional[Input[str]]) -> None:
        set_field(self, "s3_bucket", value)

    @property
    def s3_key(self) -> Optional[Input[str]]:
        return get_field(self, "s3_key")

    @s3_key.setter
    def s3_key(self, value: Optional[Input[str]]) -> None:
        set_field(self, "s3_key", value)

    @property
    def s3_object_version(self) -> Optional[Input[str]]:
        return get_field(self, "s3_object_version")

    @s3_object_version.setter
    def s3_object_version(self, value: Optional[Input[str]]) -> None:
        set_field(self, "s3_object_version", value)

    @property
    def image_uri(self) -> Optional[Input[str]]:
        return get_field(self, "image_uri")

    @image_uri.setter
    def image_uri(self, value: Optional[Input[str]]) -> None:
        set_field(self, "image_uri", value)


class Function(Construct):

    @overload
    def __init__(
        self, name: str, args: FunctionArgs, opts: Optional[ConstructOptions] = None
    ): ...

    @overload
    def __init__(
        self,
        name: str,
        handler: Optional[Input[str]] = None,
        runtime: Optional[Input[str]] = None,
        timeout: Optional[Input[int]] = None,
        memory_size: Optional[Input[int]] = None,
        environment_variables: Optional[MappingInput[str]] = None,
        code: Optional[Input[str]] = None,
        s3_bucket: Optional[Input[str]] = None,
        s3_key: Optional[Input[str]] = None,
        s3_object_version: Optional[Input[str]] = None,
        image_uri: Optional[Input[str]] = None,
        opts: Optional[ConstructOptions] = None,
    ): ...

    def __init__(self, name: str, *args, **kwargs):
        construct_args, opts = get_construct_args_opts(FunctionArgs, *args, **kwargs)
        if construct_args is not None:
            self._internal_init(name, opts=opts, **construct_args.__dict__)
        else:
            self._internal_init(name, *args, **kwargs)

    def _internal_init(
        self,
        name: str,
        handler: Optional[Input[str]] = None,
        runtime: Optional[Input[str]] = None,
        timeout: Optional[Input[int]] = None,
        memory_size: Optional[Input[int]] = None,
        environment_variables: Optional[MappingInput[str]] = None,
        code: Optional[Input[str]] = None,
        s3_bucket: Optional[Input[str]] = None,
        s3_key: Optional[Input[str]] = None,
        s3_object_version: Optional[Input[str]] = None,
        image_uri: Optional[Input[str]] = None,
        opts: Optional[ConstructOptions] = None,
    ):
        super().__init__(
            name,
            construct_type="klotho.aws.Function",
            properties={
                "Handler": handler,
                "Runtime": runtime,
                "Timeout": timeout,
                "MemorySize": memory_size,
                "EnvironmentVariables": (
                    Output.from_mapping(environment_variables)
                    if environment_variables
                    else None
                ),
                "Code": code,
                "S3Bucket": s3_bucket,
                "S3Key": s3_key,
                "S3ObjectVersion": s3_object_version,
                "ImageUri": image_uri,
            },
            opts=opts,
        )

    @property
    def function_arn(self) -> Output[str]:
        return get_output(self, "FunctionArn", str)

    @property
    def function_name(self) -> Output[str]:
        return get_output(self, "FunctionName", str)
