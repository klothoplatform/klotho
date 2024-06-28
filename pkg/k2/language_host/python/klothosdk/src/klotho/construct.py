from typing import Optional, Type, TypeVar, Generic, Union, Any

from klotho.output import Output, Input
from klotho.runtime import instance as runtime
from klotho.urn import URN

BindingType = Union["Binding", "Construct"]


class ConstructOptions:
    """
    This class is a placeholder for when we need to add common options to constructs.
    """


class Construct:
    def __init__(
        self,
        name,
        construct_type,
        properties: dict[str, Any],
        bindings: Optional[list[BindingType]] = None,
        opts: Optional[ConstructOptions] = None,
    ):
        if runtime.application is None:
            raise Exception(
                "Cannot create a construct without an application. "
                "Initialize the application first by calling klotho.Application()."
            )

        self.urn = URN(
            account_id="my-account-id",  # TODO: Get this from the runtime or environment
            project=runtime.application.project,
            application=runtime.application.name,
            environment=runtime.application.environment,
            type="construct",
            subtype=construct_type,
            resource_id=name,
        )
        self.name = name
        self.construct_type = construct_type
        self.inputs = {}
        self.outputs = {}
        self.pulumi_stack = None
        self.version = 1
        self.status = "new"  # Default status
        self.bindings: list["Binding"] = []
        self.options = opts.__dict__ if opts else {}
        self.depends_on: set[URN] = set()
        runtime.add_construct(self)
        for k, v in properties.items():
            self.add_input(k, v)

        for binding in bindings or []:
            add_binding(self, binding)

    def to_dict(self):
        data = {
            "type": self.construct_type,
            "urn": str(self.urn),
            "version": self.version,
            "pulumi_stack": self.pulumi_stack,
            "status": self.status,
            "inputs": self.inputs,
            "outputs": self.outputs,
            "bindings": [b.to_dict() for b in self.bindings],
            "options": self.options,
            "dependsOn": [str(d) for d in self.depends_on],
        }
        return {k: v for k, v in data.items() if v}

    def add_input(self, name: str, value: Any):
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
                self.inputs[name] = {"value": value, "status": "resolved"}


def get_construct_args_opts(
    construct_args_type: Type, *args, **kwargs
) -> tuple[Any, ConstructOptions]:
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

    # If opts is None, see if "opts" is in kwargs, and, if so, if it's an instance of ConstructOptions.
    if opts is None:
        o = kwargs.get("opts")
        opts = o if isinstance(o, ConstructOptions) else ConstructOptions()

    return construct_args, opts


TO = TypeVar("TO", bound=Construct)


class Binding(Generic[TO]):
    def __init__(self, to: TO, inputs: Optional[dict[str, Input[Any]]]):
        self._to: URN = to.urn
        self._inputs: dict[str, dict[str, Any]] = {}
        for k, v in inputs.items() if inputs else {}:
            self.add_input(k, v)

    def to_dict(self):
        return {"urn": str(self._to), "inputs": self._inputs}

    @property
    def to(self):
        return self._to

    @property
    def inputs(self):
        return {**self._inputs}

    def add_input(self, name: str, value: Any):
        if value is not None:
            if isinstance(value, Construct):
                self._inputs[name] = {
                    "status": "pending",
                    "dependsOn": str(value.urn),
                }
            elif isinstance(value, Output):
                self._inputs[name] = {
                    "value": None,
                    "status": "pending",
                    "dependsOn": str(value.id),
                }

            else:
                self._inputs[name] = {"value": value, "status": "resolved"}


def add_binding(source: Construct, binding: BindingType):
    # If the binding is a Construct, wrap it in a Binding.
    if isinstance(binding, Construct):
        binding = Binding(binding, {})

    if binding.to == source.urn:
        raise ValueError("Cannot bind a construct to itself.")

    # Replace any existing binding to the same construct.
    source.bindings = [b for b in source.bindings if b.to != binding.to]
    source.bindings.append(binding)

    source.depends_on.add(binding.to)

    # Add any input construct dependencies from the binding to the source.
    for v in binding.inputs.values():
        for dep in {*v.get("dependsOn", [])}:
            try:
                urn = URN.parse(dep)
                urn.output = ""
                source.depends_on.add(urn)
            except ValueError:
                # Ignore non-URN dependencies.
                pass
