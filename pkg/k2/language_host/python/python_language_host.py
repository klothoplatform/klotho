import runpy
import signal
from concurrent import futures

import grpc
import service_pb2
import service_pb2_grpc
import yaml
from klotho.runtime import instance as runtime


class KlothoService(service_pb2_grpc.KlothoServiceServicer):

    def SendIR(self, request, context):
        infra_script = request.filename
        runpy.run_path(infra_script, run_name="__main__")

        response = service_pb2.IRReply(
            message="Script executed", yaml_payload=runtime.generate_yaml()
        )
        return response

    def HealthCheck(self, request, context):
        return service_pb2.HealthCheckReply(status="Server is running!!")

    def RegisterConstruct(self, request, context):
        resources = yaml.safe_load(request.yaml_payload)
        resolved_outputs = runtime.resolve_output_references(resources)
        resolved_outputs = [
            {"id": o.id, "yaml_payload": yaml.safe_dump(o.value)}
            for o in resolved_outputs
        ]
        return service_pb2.RegisterConstructReply(
            message="Resource registered successfully",
            resolved_outputs=resolved_outputs,
        )


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    service_pb2_grpc.add_KlothoServiceServicer_to_server(KlothoService(), server)
    port = server.add_insecure_port("127.0.0.1:0")
    server.start()
    # NOTE the following print is used to communicate to the go CLI the port the server is listening on
    # The format must be kept in sync with the address parsing logic and the flush=True ensures that it is written
    # to stdout immediately, otherwise the CLI will timeout waiting for the address to be printed.
    print(f"Listening on 127.0.0.1:{port}", flush=True)
    return server


if __name__ == "__main__":
    srv = serve()

    def signal_handler(sig, frame):
        print("Termination signal received, shutting down...")
        srv.stop(0)

    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    srv.wait_for_termination()
