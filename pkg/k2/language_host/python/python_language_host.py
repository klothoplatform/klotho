import grpc
import service_pb2_grpc
import logging
import runpy
import sys
import io
import contextlib
from klotho import get_klotho

def run(infra_script):
    logging.basicConfig(level=logging.INFO)
    logger = logging.getLogger(__name__)
    channel = grpc.insecure_channel('localhost:50051')
    stub = service_pb2_grpc.KlothoServiceStub(channel)

    # Capture stdout and stderr
    stdout_capture = io.StringIO()
    stderr_capture = io.StringIO()
    with contextlib.redirect_stdout(stdout_capture), contextlib.redirect_stderr(stderr_capture):
        # Run the infra script
        runpy.run_path(infra_script, run_name="__main__")

    # Print captured stdout and stderr
    captured_stdout = stdout_capture.getvalue()
    captured_stderr = stderr_capture.getvalue()
    logger.info("Captured stdout from infra script:")
    logger.info(captured_stdout)
    if captured_stderr:
        logger.error("Captured stderr from infra script:")
        logger.error(captured_stderr)

    # Send IR after running the infra script
    
    klotho = get_klotho()
    klotho.send_ir()

if __name__ == '__main__':
    if len(sys.argv) != 2:
        print("Usage: python_language_host.py /path/to/infra.py")
        sys.exit(1)
    
    infra_script = sys.argv[1]
    run(infra_script)
