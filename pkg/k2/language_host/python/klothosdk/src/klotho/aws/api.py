from typing import Optional

from klotho.construct import Binding, Construct, ConstructOptions, add_binding


class RouteArgs:
    def __init__(self, path: str, method: str = "ANY", proxy: bool = False):
        self.path = path
        self.method = method
        self.proxy = proxy


class Api(Construct):
    def __init__(self, name: str, opts: Optional[ConstructOptions] = None):
        super().__init__(
            name, construct_type="klotho.aws.Api", properties={}, opts=opts
        )

    def route(self, routes: list[RouteArgs], destination: Construct):

        add_binding(
            self,
            Binding(
                destination,
                {
                    "Routes": [
                        {
                            "Path": route.path,
                            "Method": route.method,
                            "Proxy": route.proxy,
                        }
                        for route in routes
                    ]
                },
            ),
        )
