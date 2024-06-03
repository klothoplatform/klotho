import yaml
import grpc
import service_pb2
import service_pb2_grpc
import logging
import atexit
import threading

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
            channel = grpc.insecure_channel('localhost:50051')
            self.stub = service_pb2_grpc.KlothoServiceStub(channel)
            self._initialized = True

    def add_resource(self, resource):
        self.resources[resource.name] = resource

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

    def send_ir(self):
        logging.basicConfig(level=logging.INFO)
        logger = logging.getLogger(__name__)
        yaml_payload = self.generate_yaml()
        logger.info("Sending SendIR request...")
        response = self.stub.SendIR(service_pb2.IRRequest(error=False, yaml_payload=yaml_payload))
        logger.info(f"Received response: {response.message}")

# Singleton instance
klotho = KlothoSDK()

class Resource:
    def __init__(self, name, resource_type):
        self.name = name
        self.resource_type = resource_type
        self.inputs = {}
        self.outputs = {}
        self.pulumi_stack = None
        self.version = 1
        print("here?")
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
