from typing import Optional, overload

from klotho.construct import Construct, ConstructOptions, get_construct_args_opts
from klotho.output import Input
from klotho.type_util import set_field, get_field


class NetworkArgs:
    def __init__(self, name: Input[str]):
        set_field(self, "name", name)

    @property
    def name(self) -> Input[str]:
        return get_field(self, "name")

    @name.setter
    def name(self, value: Input[str]) -> None:
        set_field(self, "name", value)


class Network(Construct):

    @overload
    def __init__(self, args: NetworkArgs, opts: Optional[ConstructOptions] = None): ...

    @overload
    def __init__(self, name: Input[str], opts: Optional[ConstructOptions] = None): ...

    def __init__(self, *args, **kwargs):
        construct_args, opts = get_construct_args_opts(NetworkArgs, *args, **kwargs)
        if construct_args is not None:
            self._internal_init(opts, **construct_args.__dict__)
        else:
            self._internal_init(*args, **kwargs)

    def _internal_init(
        self,
        name: Input[str],
        opts: Optional[ConstructOptions] = None,
    ):
        super().__init__(
            name,
            construct_type="klotho.aws.Network",
            properties={},
            opts=opts,
        )
