import argparse
import runpy
import signal
import sys
from concurrent import futures

import grpc
import yaml

import service_pb2
import service_pb2_grpc
from debug_util import DebugMode, configure_debugging
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
        resolved_outputs = yaml.safe_dump(runtime.resolve_output_references(resources))
        return service_pb2.RegisterConstructReply(
            message="Resource registered successfully", yaml_payload=resolved_outputs)


def serve() -> grpc.Server:
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    service_pb2_grpc.add_KlothoServiceServicer_to_server(KlothoService(), server)
    port = server.add_insecure_port("127.0.0.1:0")
    if port == 0:
        raise Exception('Failed to bind to port')
    print(f'Starting server on port {port}')
    server.start()
    # NOTE the following print is used to communicate to the go CLI the port the server is listening on
    # The format must be kept in sync with the address parsing logic and the flush=True ensures that it is written
    # to stdout immediately, otherwise the CLI will timeout waiting for the address to be printed.
    print(f"Listening on 127.0.0.1:{port}", flush=True)
    return server


# simple cli to start the server with args
def cli():
    ap = argparse.ArgumentParser()
    ap.add_argument("--debug", type=lambda c: DebugMode[c.upper()], default=DebugMode.NONE, help="Enable debugging")
    ap.add_argument("--debug-port", type=int, default=5678, help="Port to run the debugger on")

    args = ap.parse_args()

    if args.debug != DebugMode.NONE:
        configure_debugging(args.debug_port, args.debug)

    srv = serve()

    def signal_handler(sig, frame):
        print("Termination signal received, shutting down...")
        srv.stop(0)

    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    srv.wait_for_termination()


if __name__ == "__main__":
    cli()
