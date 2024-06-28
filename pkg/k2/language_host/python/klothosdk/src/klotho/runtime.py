import threading
from typing import Optional, TYPE_CHECKING, Any

import grpc
import yaml

import klotho
from klotho import service_pb2_grpc as service_pb2_grpc
from klotho.provider import Provider
from klotho.urn import URN

if TYPE_CHECKING:
    from klotho.output import Output


class Runtime:
    _instance = None
    _lock = threading.Lock()

    def __new__(cls):
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None:
                    cls._instance = super(Runtime, cls).__new__(cls)
                    cls._instance._initialized = False
        return cls._instance

    def __init__(self):
        if not self._initialized:
            self.constructs = {}
            self.outputs: dict[str, Any] = {}
            self.output_references: dict[str, "Output"] = {}
            self.application: Optional["klotho.Application"] = None
            channel = grpc.insecure_channel("localhost:50051")
            self.stub = service_pb2_grpc.KlothoServiceStub(channel)
            self._initialized = True
            self.providers: dict[str, Provider] = {}

    def add_construct(self, resource):
        self.constructs[resource.name] = resource

    def set_application(self, application):
        self.application = application

    def generate_yaml(self):
        constructs = {
            name: construct.to_dict() for name, construct in self.constructs.items()
        }
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

    def resolve_output_references(
        self, constructs: dict[str, dict[str, Any]]
    ) -> dict[str, Any]:
        """
        constructs is expected to be a dictionary of resource urn to resource outputs
        example:
        {
            "urn:accountid:project:env:construct:...": {
                "output1": "value1",
                "output2": "value2"
            }
        }

        :return: a dictionary of resolved output references where the key is the output id and the value is the resolved value
        """
        for urn, outputs in constructs.items():
            for output_name, output_value in outputs.items():
                output_urn = URN.parse(urn)
                output_urn.output = output_name
                self.outputs[str(output_urn)] = output_value

        remaining_unresolved_outputs = {
            output_id: output
            for output_id, output in self.output_references.items()
            if not output.is_resolved
        }
        resolved_output_references = {}
        resolved_count = -1
        while resolved_count != 0:
            resolved_count = 0
            unresolved_outputs = remaining_unresolved_outputs
            remaining_unresolved_outputs = {}
            for output_id, output in unresolved_outputs.items():
                resolved_deps = []
                for dep in output.depends_on:
                    if dep in self.outputs:
                        resolved_deps.append(self.outputs[dep])
                if len(resolved_deps) != len(output.depends_on):
                    remaining_unresolved_outputs[output_id] = output
                    continue
                output.resolve(resolved_deps)
                resolved_count += 1
                resolved_output_references[output_id] = output.value
        return resolved_output_references

    def set_provider(self, name: str, provider: Provider):
        self.providers[name] = provider


instance: Runtime = Runtime()
