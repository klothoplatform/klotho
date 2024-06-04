import grpc
import service_pb2_grpc, service_pb2
import threading
import logging
import yaml

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
            "app_urn": f"urn:accountid:{self.application.project}:{self.application.environment}:{self.application.name}",
            "project_urn": f"urn:accountid:{self.application.project}",
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
