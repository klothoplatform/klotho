import enum


class DebugMode(enum.Enum):
    NONE = 0
    VSCODE = 1
    INTELLIJ = 2

def configure_debugging(port: int, mode: DebugMode = DebugMode.VSCODE):
    if mode == DebugMode.VSCODE:
        import debugpy

        debugpy.listen(("localhost", port))
        print("Waiting for debugger attach...")

        debugpy.wait_for_client()
        print("Debugger attached.")
    elif mode == DebugMode.INTELLIJ:
        import pydevd_pycharm
        pydevd_pycharm.settrace('localhost', port=port, stdoutToServer=True, stderrToServer=True, suspend=False)
        print("Debugger attached.")

