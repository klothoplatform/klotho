import grpc
import service_pb2
import service_pb2_grpc
import time
import logging

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def run():
    logger.info("Python client starting...")
    with grpc.insecure_channel('localhost:50051') as channel:
        stub = service_pb2_grpc.ExampleServiceStub(channel)
        
        # SayHello call
        logger.info("Sending SayHello request...")
        response = stub.SayHello(service_pb2.HelloRequest(name='World'))
        logger.info(f"SayHello Response: {response.message}")
        
        # GetPythonResponse call
        logger.info("Sending GetPythonResponse request...")
        response = stub.GetPythonResponse(service_pb2.PythonRequest(query='Python query'))
        logger.info(f"GetPythonResponse Response: {response.response}")

if __name__ == '__main__':
    run()
