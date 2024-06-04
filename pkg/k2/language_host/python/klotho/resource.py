class Resource:
    def __init__(self, name, resource_type):
        self.name = name
        self.resource_type = resource_type
        self.inputs = {}
        self.outputs = {}
        self.pulumi_stack = None
        self.version = 1
        self.status = "new"  # Default status
        self.bindings = []
        self.options = {}
        self.depends_on = []
        from . import get_klotho
        klotho = get_klotho()
        klotho.add_resource(self)

    def to_dict(self, application):
        data = {
            "type": self.resource_type,
            "urn": f"urn:accountid:{application.project}:{application.environment}::construct/{self.resource_type}:{self.name}",
            "version": self.version,
            "pulumi_stack": self.pulumi_stack,
            "status": self.status,
            "inputs": self.inputs,
            "outputs": self.outputs,
            "bindings": self.bindings,
            "options": self.options,
            "dependsOn": self.depends_on,
        }
        return {k: v for k, v in data.items() if v}

    def add_input(self, name, input_type, value):
        if value is not None:
            self.inputs[name] = {
                "type": input_type,
                "value": value
            }
