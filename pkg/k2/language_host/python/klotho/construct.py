from typing import Optional

from klotho.output import Output
from klotho.runtime import instance as runtime
from klotho.urn import URN


class ConstructOptions:
    """
    This class is a placeholder for when we need to add common options to constructs.
    """


class Construct:
    def __init__(self, name, construct_type, properties: dict[str, any], opts: Optional[ConstructOptions] = None):
        self.urn = URN(
            account_id="my-account-id",  # TODO: Get this from the runtime or environment
            project=runtime.application.project,
            application=runtime.application.name,
            environment=runtime.application.environment,
            type="construct",
            subtype=construct_type,
            resource_id=name
        )
        self.name = name
        self.construct_type = construct_type
        self.inputs = {}
        self.outputs = {}
        self.pulumi_stack = None
        self.version = 1
        self.status = "new"  # Default status
        self.bindings = []
        self.options = opts.__dict__ if opts else {}
        self.depends_on: set[URN] = set()
        runtime.add_construct(self)
        for k, v in properties.items():
            self.add_input(k, v)

    def to_dict(self):
        data = {
            "type": self.construct_type,
            "urn": str(self.urn),
            "version": self.version,
            "pulumi_stack": self.pulumi_stack,
            "status": self.status,
            "inputs": self.inputs,
            "outputs": self.outputs,
            "bindings": self.bindings,
            "options": self.options,
            "dependsOn": [str(d) for d in self.depends_on],
        }
        return {k: v for k, v in data.items() if v}

    def add_input(self, name: str, value: any):
        if value is not None:
            if isinstance(value, Construct):
                self.depends_on.add(value.urn)
                self.inputs[name] = {
                    "status": "pending",
                    "dependsOn": str(value.urn),
                }
            elif isinstance(value, Output):
                self.inputs[name] = {
                    "value": None,
                    "status": "pending",
                    "dependsOn": str(value.id),
                }

                for dep in {*value.depends_on, value.id}:
                    try:
                        # If the dep is a URN, add it to the list of dependencies (make sure to strip the output name).
                        urn = URN.parse(dep)
                        urn.output = ""
                        self.depends_on.add(urn)
                    except ValueError:
                        # If the dep is not a URN, it's not a construct dependency, so we can ignore it.
                        pass

            else:
                self.inputs[name] = {
                    "value": value,
                    "status": "resolved"
                }


def get_construct_args_opts(construct_args_type, *args, **kwargs):
    """
    Return the construct args and options given the *args and **kwargs of a construct's
    __init__ method.
    """

    construct_args, opts = None, None

    # If the first item is the construct args type, save it and remove it from the args list.
    if args and isinstance(args[0], construct_args_type):
        construct_args, args = args[0], args[1:]

    # Now look at the first item in the args list again.
    # If the first item is the construct options class, save it.
    if args and isinstance(args[0], ConstructOptions):
        opts = args[0]

    # If construct_args is None, see if "args" is in kwargs, and, if so, if it's typed as the
    # construct args type.
    if construct_args is None:
        a = kwargs.get("args")
        if isinstance(a, construct_args_type):
            construct_args = a

    # If opts is None, look it up in kwargs.
    if opts is None:
        opts = kwargs.get("opts")

    return construct_args, opts
