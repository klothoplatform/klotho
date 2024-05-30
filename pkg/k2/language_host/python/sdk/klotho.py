import yaml

class KlothoSDK:
    def __init__(self):
        self.resources = {}

    def add_resource(self, resource):
        self.resources[resource.name] = resource

    # this will be something that's called at the very end and will make the
    # yaml file that will be sent to the Klotho service via the gRPC call
    def generate_yaml(self):
        constructs = {name: resource.to_dict() for name, resource in self.resources.items()}
        output = {
            "schemaVersion": 1,
            "version": 1,
            "project_urn": "urn:accountid:project",
            "app_urn": "urn:accountid:project:application::my-app",
            "environment": "dev",
            "constructs": constructs,
        }
        return yaml.dump(output, sort_keys=False)
    
klotho = KlothoSDK()

class Resource:
    def __init__(self, name, resource_type):
        self.name = name
        self.resource_type = resource_type
        self.inputs = {}
        self.outputs = {}
        self.pulumi_stack = None
        self.version = 1
        klotho.add_resource(self)

    def to_dict(self):
        return {
            "type": self.resource_type,
            "urn": f"urn:accountid:my-project:dev:construct/{self.resource_type}:{self.name}",
            "version": self.version,
            "pulumi_stack": self.pulumi_stack,
            "inputs": self.inputs,
            "outputs": self.outputs,
        }
    
class Container(Resource):
    def __init__(self, name, image):
        super().__init__(name, "klotho.aws.Container")
        self.inputs["image"] = {
            "type": "string",
            "value": image
        }
    

