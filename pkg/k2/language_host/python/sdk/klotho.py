import yaml
import grpc
import service_pb2
import service_pb2_grpc
import logging
import threading
import os

class KlothoSDK:
    _instance = None
    _lock = threading.Lock()

    def __new__(cls):
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None:
                    cls._instance = super(KlothoSDK, cls).__new__(cls)
                    cls._instance._initialized = False
        return cls._instance

    def __init__(self):
        if not self._initialized:
            self.resources = {}
            self.application = None
            channel = grpc.insecure_channel('localhost:50051')
            self.stub = service_pb2_grpc.KlothoServiceStub(channel)
            self._initialized = True

    def add_resource(self, resource):
        self.resources[resource.name] = resource

    def set_application(self, application):
        self.application = application

    def generate_yaml(self):
        constructs = {name: resource.to_dict(self.application) for name, resource in self.resources.items()}
        output = {
            "schemaVersion": 1,  # Adjust this value as needed
            "version": 1,  # Adjust this value as needed
            "urn": f"urn:accountid:{self.application.project}:{self.application.environment}:{self.application.name}",
            "project": self.application.project,
            "environment": self.application.environment,
            "default_region": self.application.default_region,
            "constructs": constructs,
        }
        return yaml.dump(output, sort_keys=False)

    def send_ir(self):
        logging.basicConfig(level=logging.INFO)
        logger = logging.getLogger(__name__)
        yaml_payload = self.generate_yaml()
        logger.info("Sending SendIR request...")
        response = self.stub.SendIR(service_pb2.IRRequest(error=False, yaml_payload=yaml_payload))
        logger.info(f"Received response: {response.message}")

# Singleton instance
klotho = KlothoSDK()

class Application:
    def __init__(self, name, project=None, environment=None, default_region=None):
        self.name = name
        self.project = project or os.getenv('PROJECT_NAME', 'default')
        self.environment = environment or os.getenv('KLOTHO_ENVIRONMENT', 'default')
        self.default_region = default_region or os.getenv('AWS_REGION', 'us-east-1')
        klotho.set_application(self)

# Resource serializes to our IR format
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
        klotho.add_resource(self)

    def to_dict(self, application):
        data = {
            "type": self.resource_type,
            "urn": f"urn:accountid:{application.project}:{application.environment}:construct/{self.resource_type}:{self.name}",
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

class Container(Resource):
    def __init__(self, name, image, source_hash=None, cpu=256, memory=512, context=None, dockerfile=None, port=None, network=None):
        super().__init__(name, "klotho.aws.Container")
        self.add_input("SourceHash", "string", source_hash)
        self.add_input("Cpu", "number", cpu)
        self.add_input("Memory", "number", memory)
        self.add_input("Context", "string", context)
        self.add_input("Dockerfile", "string", dockerfile)
        self.add_input("Image", "string", image)
        self.add_input("Port", "number", port)
        self.add_input("Network", "Construct<klotho.aws.Network>", network)

        # Add the container to the KlothoSDK resource list
        klotho.add_resource(self)
