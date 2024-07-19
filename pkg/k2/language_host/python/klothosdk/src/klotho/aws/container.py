from typing import Optional, overload, Union, TYPE_CHECKING

from klotho.aws.network import Network
from klotho.construct import (
    ConstructOptions,
    get_construct_args_opts,
    Construct,
    Binding,
    add_binding,
)
from klotho.output import Input, Output, MappingInput
from klotho.runtime_util import get_default_construct
from klotho.type_util import set_field, get_field, get_output

if TYPE_CHECKING:
    from klotho.aws.bucket import Bucket
    from klotho.aws.postgres import Postgres

BindingType = Union[
    Binding["Bucket"], "Bucket",
    Binding["Postgres"], "Postgres",
]


class ContainerArgs:
    def __init__(
        self,
        bindings: Optional[list[BindingType]] = None,
        context: Optional[Input[str]] = None,
        cpu: Optional[Input[int]] = None,
        dockerfile: Optional[Input[str]] = None,
        enable_execute_command: Optional[Input[bool]] = None,
        environment_variables: Optional[MappingInput[str]] = None,
        image: Optional[Input[str]] = None,
        memory: Optional[Input[int]] = None,
        network: Optional[Network] = None,
        port: Optional[Input[int]] = None,
    ):
        if bindings is not None:
            set_field(self, "bindings", bindings)
        if context is not None:
            set_field(self, "context", context)
        if cpu is not None:
            set_field(self, "cpu", cpu)
        if dockerfile is not None:
            set_field(self, "dockerfile", dockerfile)
        if enable_execute_command is not None:
            set_field(self, "enable_execute_command", enable_execute_command)
        if environment_variables is not None:
            set_field(self, "environment_variables", environment_variables)
        if image is not None:
            set_field(self, "image", image)
        if memory is not None:
            set_field(self, "memory", memory)
        if network is not None:
            set_field(self, "network", network)
        if port is not None:
            set_field(self, "port", port)

    @property
    def image(self) -> Input[str] | None:
        return get_field(self, "image")

    @image.setter
    def image(self, value: Input[str]) -> None:
        set_field(self, "image", value)

    @property
    def cpu(self) -> Optional[Input[int]]:
        return get_field(self, "cpu")

    @cpu.setter
    def cpu(self, value: Optional[Input[int]]) -> None:
        set_field(self, "cpu", value)

    @property
    def memory(self) -> Optional[Input[int]]:
        return get_field(self, "memory")

    @memory.setter
    def memory(self, value: Optional[Input[int]]) -> None:
        set_field(self, "memory", value)

    @property
    def context(self) -> Optional[Input[str]]:
        return get_field(self, "context")

    @context.setter
    def context(self, value: Optional[Input[str]]) -> None:
        set_field(self, "context", value)

    @property
    def dockerfile(self) -> Optional[Input[str]]:
        return get_field(self, "dockerfile")

    @dockerfile.setter
    def dockerfile(self, value: Optional[Input[str]]) -> None:
        set_field(self, "dockerfile", value)

    @property
    def network(self) -> Network | None:
        return get_field(self, "network")

    @network.setter
    def network(self, value: Optional[Network]) -> None:
        set_field(self, "network", value)

    @property
    def enable_execute_command(self) -> Optional[Input[bool]]:
        return get_field(self, "enable_execute_command")

    @enable_execute_command.setter
    def enable_execute_command(self, value: Optional[Input[bool]]) -> None:
        set_field(self, "enable_execute_command", value)

    @property
    def environment_variables(self) -> Optional[MappingInput[str]]:
        return get_field(self, "environment_variables")

    @environment_variables.setter
    def environment_variables(self, value: Optional[MappingInput[str]]) -> None:
        set_field(self, "environment_variables", value)

    @property
    def port(self) -> Optional[Input[int]]:
        return get_field(self, "port")

    @port.setter
    def port(self, value: Optional[Input[int]]) -> None:
        set_field(self, "port", value)


class Container(Construct):

    @overload
    def __init__(
        self, name: str, args: ContainerArgs, opts: Optional[ConstructOptions] = None
    ): ...

    @overload
    def __init__(
        self,
        name: str,
        bindings: Optional[list[BindingType]] = None,
        context: Optional[Input[str]] = None,
        cpu: Optional[Input[int]] = None,
        dockerfile: Optional[Input[str]] = None,
        enable_execute_command: Optional[Input[bool]] = None,
        environment_variables: Optional[MappingInput[str]] = None,
        image: Optional[Input[str]] = None,
        memory: Optional[Input[int]] = None,
        network: Optional[Network] = None,
        port: Optional[Input[int]] = None,
        opts: Optional[ConstructOptions] = None,
    ): ...

    def __init__(self, name: str, *args, **kwargs):
        construct_args, opts = get_construct_args_opts(ContainerArgs, *args, **kwargs)
        if construct_args is not None:
            self._internal_init(name, opts=opts, **construct_args.__dict__)
        else:
            self._internal_init(name, *args, **kwargs)

    def _internal_init(
        self,
        name: str,
        context: Optional[Input[str]] = None,
        cpu: Optional[Input[int]] = None,
        dockerfile: Optional[Input[str]] = None,
        enable_execute_command: Optional[Input[bool]] = None,
        environment_variables: Optional[MappingInput[str]] = None,
        image: Optional[Input[str]] = None,
        memory: Optional[Input[int]] = None,
        network: Optional[Network] = None,
        port: Optional[Input[int]] = None,
        bindings: Optional[list[BindingType]] = None,
        opts: Optional[ConstructOptions] = None,
    ):
        if network is None:
            network = get_default_construct("aws", Network)

        super().__init__(
            name,
            construct_type="klotho.aws.Container",
            properties={
                "Context": context,
                "Cpu": cpu,
                "Dockerfile": dockerfile,
                "EnableExecuteCommand": enable_execute_command,
                "EnvironmentVariables": (
                    Output.from_mapping(environment_variables)
                    if environment_variables
                    else None
                ),
                "Image": image,
                "Memory": memory,
                "Network": network,
                "Port": port,
            },
            bindings=bindings,
            opts=opts,
        )

    @property
    def load_balancer_url(self) -> Output[str]:
        return get_output(self, "LoadBalancerUrl", str)

    def bind(self, binding: BindingType) -> None:
        add_binding(self, binding)
