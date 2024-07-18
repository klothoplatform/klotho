from typing import Optional, overload, Union, TYPE_CHECKING

from klotho.aws.network import Network
from klotho.construct import ConstructOptions, get_construct_args_opts, Construct, Binding, add_binding
from klotho.output import Input, Output
from klotho.runtime_util import get_default_construct
from klotho.type_util import set, get, get_output

if TYPE_CHECKING:
    pass

BindingType = Union[
    Binding["Bucket"], "Bucket"
]


class FastAPIArgs:
    def __init__(self,
                 bindings: Optional[list[BindingType]] = None,
                 context: Optional[Input[str]] = None,
                 cpu: Optional[Input[int]] = None,
                 dockerfile: Optional[Input[str]] = None,
                 enable_execute_command: Optional[Input[bool]] = None,
                 image: Optional[Input[str]] = None,
                 memory: Optional[Input[int]] = None,
                 port: Optional[Input[int]] = None,
                 network: Optional[Network] = None,
                 source_hash: Optional[Input[str]] = None,
                 ):
        if bindings is not None:
            set(self, "bindings", bindings)
        if context is not None:
            set(self, "context", context)
        if cpu is not None:
            set(self, "cpu", cpu)
        if dockerfile is not None:
            set(self, "dockerfile", dockerfile)
        if enable_execute_command is not None:
            set(self, "enable_execute_command", enable_execute_command)
        if image is not None:
            set(self, "image", image)
        if memory is not None:
            set(self, "memory", memory)
        if network is not None:
            set(self, "network", network)
        if port is not None:
            set(self, "port", port)
        if source_hash is not None:
            set(self, "source_hash", source_hash)

    @property
    def image(self) -> Input[str] | None:
        return get(self, "image")

    @image.setter
    def image(self, value: Input[str]) -> None:
        set(self, "image", value)

    @property
    def source_hash(self) -> Optional[Input[str]]:
        return get(self, "source_hash")

    @source_hash.setter
    def source_hash(self, value: Optional[Input[str]]) -> None:
        set(self, "source_hash", value)

    @property
    def cpu(self) -> Optional[Input[int]]:
        return get(self, "cpu")

    @cpu.setter
    def cpu(self, value: Optional[Input[int]]) -> None:
        set(self, "cpu", value)

    @property
    def memory(self) -> Optional[Input[int]]:
        return get(self, "memory")

    @memory.setter
    def memory(self, value: Optional[Input[int]]) -> None:
        set(self, "memory", value)

    @property
    def context(self) -> Optional[Input[str]]:
        return get(self, "context")

    @context.setter
    def context(self, value: Optional[Input[str]]) -> None:
        set(self, "context", value)

    @property
    def dockerfile(self) -> Optional[Input[str]]:
        return get(self, "dockerfile")

    @dockerfile.setter
    def dockerfile(self, value: Optional[Input[str]]) -> None:
        set(self, "dockerfile", value)

    @property
    def port(self) -> Optional[Input[int]]:
        return get(self, "port")

    @port.setter
    def port(self, value: Optional[Input[int]]) -> None:
        set(self, "port", value)

    @property
    def network(self) -> Network | None:
        return get(self, "network")

    @network.setter
    def network(self, value: Optional[Network]) -> None:
        set(self, "network", value)

    @property
    def enable_execute_command(self) -> Optional[Input[bool]]:
        return get(self, "enable_execute_command")

    @enable_execute_command.setter
    def enable_execute_command(self, value: Optional[Input[bool]]) -> None:
        set(self, "enable_execute_command", value)


class FastAPI(Construct):

    @overload
    def __init__(self, name: str, args: FastAPIArgs, opts: Optional[ConstructOptions] = None):
        ...

    @overload
    def __init__(
            self,
            name: str,
            bindings: Optional[list[BindingType]] = None,
            context: Optional[Input[str]] = None,
            cpu: Optional[Input[int]] = None,
            dockerfile: Optional[Input[str]] = None,
            enable_execute_command: Optional[Input[bool]] = None,
            image: Optional[Input[str]] = None,
            memory: Optional[Input[int]] = None,
            network: Optional[Network] = None,
            port: Optional[Input[int]] = None,
            source_hash: Optional[Input[str]] = None,
            opts: Optional[ConstructOptions] = None):
        ...

    def __init__(self, name: str, *args, **kwargs):
        construct_args, opts = get_construct_args_opts(FastAPIArgs, *args, **kwargs)
        if construct_args is not None:
            self._internal_init(name, opts=opts, **construct_args.__dict__)
        else:
            self._internal_init(name, *args, **kwargs)

    def _internal_init(
            self,
            name: str,
            image: Optional[Input[str]] = None,
            context: Optional[Input[str]] = None,
            cpu: Optional[Input[int]] = None,
            dockerfile: Optional[Input[str]] = None,
            enable_execute_command: Optional[Input[bool]] = None,
            memory: Optional[Input[int]] = None,
            network: Optional[Network] = None,
            port: Optional[Input[int]] = None,
            source_hash: Optional[Input[str]] = None,
            bindings: Optional[list[BindingType]] = None,
            opts: Optional[ConstructOptions] = None
    ):
        if network is None:
            network = get_default_construct("aws", Network)

        super().__init__(
            name,
            construct_type="klotho.aws.FastAPI",
            properties={
                "Image": image,
                "SourceHash": source_hash,
                "Cpu": cpu,
                "Memory": memory,
                "Context": context,
                "Dockerfile": dockerfile,
                "Port": port,
                "Network": network,
                "EnableExecuteCommand": enable_execute_command
            },
            bindings=bindings,
            opts=opts,
        )

    @property
    def load_balancer_url(self) -> Output[str]:
        return get_output(self, "LoadBalancerUrl", str)

    def bind(self, binding: BindingType) -> None:
        add_binding(self, binding)
