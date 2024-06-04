import grpc
import service_pb2
import service_pb2_grpc
import runpy
from concurrent import futures
from klotho import get_klotho
import signal
import sys

class KlothoService(service_pb2_grpc.KlothoServiceServicer):

    def SendIR(self, request, context):
        infra_script = request.filename
        runpy.run_path(infra_script, run_name="__main__")

        klotho = get_klotho()

        response = service_pb2.IRReply(
            message="Script executed",
            yaml_payload=klotho.generate_yaml()
        )
        return response

    def HealthCheck(self, request, context):
        return service_pb2.HealthCheckReply(status="Server is running!!")

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    service_pb2_grpc.add_KlothoServiceServicer_to_server(KlothoService(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    server.wait_for_termination()

def signal_handler(sig, frame):
    print('Termination signal received, shutting down...')
    sys.exit(0)

if __name__ == '__main__':
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    serve()
