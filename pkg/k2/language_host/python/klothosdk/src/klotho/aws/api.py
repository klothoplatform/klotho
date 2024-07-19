from typing import Optional

from klotho.construct import Binding, Construct, ConstructOptions, add_binding


class Api(Construct):
    def __init__(self, name: str, opts: Optional[ConstructOptions] = None):
        super().__init__(
            name, construct_type="klotho.aws.Api", properties={}, opts=opts
        )

    def route_to(self, path: str, dest: Construct):
        add_binding(self, Binding(dest, {"Path": path}))
